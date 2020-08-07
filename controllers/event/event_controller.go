/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package event

import (
	"context"
	"reflect"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/fatih/structs"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	eventLog = ctrl.Log.WithName("event")
)

// Reconciler reconciles a Event object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config *config
}

// TODO
// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=events/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	reqLogger := r.Log.WithValues("Namespace", req.Namespace, "Name", req.Name)
	if r.Config.name == "" {
		r.Config.name = req.Name
	}

	reqLogger.V(2).Info("Reconciling event logger")

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Config.filter = nil
			reqLogger.Info("cr was deleted, removing filter")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	needUpdate := false
	if !reflect.DeepEqual(r.Config.logFields, cr.Spec.LogFields) {
		r.Config.logFields = cr.Spec.LogFields
		reqLogger.WithValues("logFields", r.Config.logFields).Info("apply new log fields")
		needUpdate = true
	}

	newFilter := newFilter(cr.Spec)
	if r.Config.filter == nil || !r.Config.filter.Equals(newFilter) {
		r.Config.filter = newFilter
		reqLogger.WithValues("filter", r.Config.filter).Info("apply new filter")
		needUpdate = true
	}

	if needUpdate {
		return r.updateCR(ctx, cr, reqLogger, nil)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) updateCR(ctx context.Context, cr *eventloggerv1.EventLogger, logger logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		logger.Error(err, "")
	}
	cr.Apply(err)
	err = r.Update(ctx, cr)
	return reconcile.Result{}, err
}

type loggingPredicate struct {
	predicate.Funcs
	lastVersion string
	Config      *config
}

// Create implements Predicate
func (p loggingPredicate) Create(e event.CreateEvent) bool {
	return p.logEvent(e.Meta, e.Object)
}

// Delete implements Predicate
func (p loggingPredicate) Delete(e event.DeleteEvent) bool {
	return p.logEvent(e.Meta, e.Object)
}

// Update implements Predicate
func (p loggingPredicate) Update(e event.UpdateEvent) bool {
	return p.logEvent(e.MetaNew, e.ObjectNew)
}

func (p loggingPredicate) logEvent(mo metav1.Object, e runtime.Object) bool {
	if p.Config == nil || p.Config.filter == nil {
		return false
	}

	evt := e.(*corev1.Event)
	if evt.ResourceVersion <= p.lastVersion {
		return false
	}
	p.lastVersion = evt.ResourceVersion

	if p.shouldLog(evt) {
		var eventLogger logr.Logger
		if len(p.Config.logFields) == 0 {
			eventLogger = eventLog.WithValues(
				"namespace", evt.ObjectMeta.Namespace,
				"name", evt.ObjectMeta.Name,
				"reason", evt.Reason,
				"timestamp", evt.LastTimestamp,
				"type", evt.Type,
				"involvedObject ", evt.InvolvedObject,
				"source ", evt.Source,
			)
		} else {
			m := structs.Map(evt)
			eventLogger = eventLog
			for _, lf := range p.Config.logFields {
				if len(lf.Path) > 0 {
					val, ok, err := unstructured.NestedFieldNoCopy(m, lf.Path...)
					if ok && err == nil {
						eventLogger = eventLogger.WithValues(lf.Name, val)
					}
				}
			}
		}

		eventLogger.Info(evt.Message)
	}
	return false
}

func (p *loggingPredicate) shouldLog(e *corev1.Event) bool {

	if len(p.Config.filter.Kinds) == 0 {
		return len(p.Config.filter.EventTypes) == 0 || p.contains(p.Config.filter.EventTypes, e.Type)
	}

	k, ok := p.Config.filter.Kinds[e.InvolvedObject.Kind]
	if !ok {
		return false
	}

	if len(k.EventTypes) != 0 && !p.contains(k.EventTypes, e.Type) {
		return false
	}

	return p.matches(k.MatchingPatterns, k.SkipOnMatch, e.Message)
}

func (p *loggingPredicate) matches(patterns []*regexp.Regexp, skipOnMatch bool, val string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if p.MatchString(val) {
			return !skipOnMatch
		}
	}
	return skipOnMatch
}

func (p *loggingPredicate) contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

func getLatestRevision(ctx context.Context, mgr manager.Manager, namespace string) (string, error) {

	cl, err := client.New(mgr.GetConfig(), client.Options{})
	if err != nil {
		return "", err
	}

	eventList := &corev1.EventList{}
	opts := []client.ListOption{
		client.Limit(0),
		client.InNamespace(namespace),
	}

	err = cl.List(ctx, eventList, opts...)
	if err != nil {
		return "", err
	}
	return eventList.ResourceVersion, nil
}

type eventLoggerPredicate struct {
	predicate.Funcs
	Config *config
}

// Create implements Predicate
func (p eventLoggerPredicate) Create(e event.CreateEvent) bool {
	return p.Config.matches(e.Meta)
}

// Update implements Predicate
func (p eventLoggerPredicate) Update(e event.UpdateEvent) bool {
	return p.Config.matches(e.MetaNew)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, namespace string) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&eventloggerv1.EventLogger{}).
		Watches(&source.Kind{Type: &eventloggerv1.EventLogger{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(eventLoggerPredicate{Config: r.Config}).
		Complete(r)

	if err != nil {
		return err
	}
	// Watch for changes to primary resource Event
	p := &loggingPredicate{Config: r.Config}
	p.lastVersion, err = getLatestRevision(context.Background(), mgr, namespace)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Watches(&source.Kind{Type: &corev1.Event{}}, &handler.Funcs{}).
		WithEventFilter(p).
		Complete(r)

}
