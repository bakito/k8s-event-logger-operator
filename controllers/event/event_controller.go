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
	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"regexp"
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

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cfg    *config
}

// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=events/status,verbs=get;update;patch

func (r *EventReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("event", req.NamespacedName)
	if r.cfg.name == "" {
		r.cfg.name = req.Name
	}

	reqLogger := r.Log.WithValues("Namespace", req.Namespace, "Name", req.Name)
	reqLogger.V(2).Info("Reconciling event logger")

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.Get(context.TODO(), req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.cfg.filter = nil
			reqLogger.Info("cr was deleted, removing filter")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return r.updateCR(cr, reqLogger, err)
	}

	needUpdate := false
	if !reflect.DeepEqual(r.cfg.logFields, cr.Spec.LogFields) {
		r.cfg.logFields = cr.Spec.LogFields
		reqLogger.WithValues("logFields", r.cfg.logFields).Info("apply new log fields")
		needUpdate = true
	}

	newFilter := newFilter(cr.Spec)
	if r.cfg.filter == nil || !r.cfg.filter.Equals(newFilter) {
		r.cfg.filter = newFilter
		reqLogger.WithValues("filter", r.cfg.filter).Info("apply new filter")
		needUpdate = true
	}

	if needUpdate {
		return r.updateCR(cr, reqLogger, nil)
	}

	return reconcile.Result{}, nil
}

func (r *EventReconciler) updateCR(cr *eventloggerv1.EventLogger, logger logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		logger.Error(err, "")
	}
	cr.Apply(err)
	err = r.Update(context.TODO(), cr)
	return reconcile.Result{}, err
}

type loggingPredicate struct {
	predicate.Funcs
	lastVersion string
	cfg         *config
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
	if p.cfg == nil || p.cfg.filter == nil {
		return false
	}

	evt := e.(*corev1.Event)
	if evt.ResourceVersion <= p.lastVersion {
		return false
	}
	p.lastVersion = evt.ResourceVersion

	if p.shouldLog(evt) {
		var eventLogger logr.Logger
		if len(p.cfg.logFields) == 0 {
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
			for _, lf := range p.cfg.logFields {
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

	if len(p.cfg.filter.Kinds) == 0 {
		return len(p.cfg.filter.EventTypes) == 0 || p.contains(p.cfg.filter.EventTypes, e.Type)
	}

	k, ok := p.cfg.filter.Kinds[e.InvolvedObject.Kind]
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

func getLatestRevision(mgr manager.Manager) (string, error) {

	cl, err := client.New(mgr.GetConfig(), client.Options{})
	if err != nil {
		return "", err
	}

	namespace, _ := k8sutil.GetWatchNamespace()
	eventList := &corev1.EventList{}
	opts := []client.ListOption{
		client.Limit(0),
		client.InNamespace(namespace),
	}

	err = cl.List(context.TODO(), eventList, opts...)
	if err != nil {
		return "", err
	}
	return eventList.ResourceVersion, nil
}

type eventLoggerPredicate struct {
	predicate.Funcs
	cfg *config
}

// Create implements Predicate
func (p eventLoggerPredicate) Create(e event.CreateEvent) bool {
	return p.cfg.matches(e.Meta)
}

// Update implements Predicate
func (p eventLoggerPredicate) Update(e event.UpdateEvent) bool {
	return p.cfg.matches(e.MetaNew)
}

func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
