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

package logging

import (
	"context"
	"reflect"

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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var eventLog = ctrl.Log.WithName("event")

// Reconciler reconciles a Event object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config *Config
	// LoggerMode if enabled, the controller does only logging and no update on the custom resource
	LoggerMode bool
}

// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=eventloggers,verbs=get;list;watch;create;update;patch;delete

// Reconcile EventLogger to update the current config
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)
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
		reqLogger.WithValues("filter", r.Config.filter.String()).Info("apply new filter")
		needUpdate = true
	}

	if needUpdate {
		return r.updateCR(ctx, cr, reqLogger, nil)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) updateCR(
	ctx context.Context,
	cr *eventloggerv1.EventLogger,
	logger logr.Logger,
	err error,
) (reconcile.Result, error) {
	if err != nil {
		logger.Error(err, "")
	}
	if r.LoggerMode {
		// return only, no update
		return reconcile.Result{}, err
	}
	cr.Apply(err)
	err = r.Update(ctx, cr)
	return reconcile.Result{}, err
}

type loggingPredicate struct {
	predicate.Funcs
	lastVersion string
	Config      *Config
}

// Create implements Predicate
func (p *loggingPredicate) Create(e event.CreateEvent) bool {
	if _, ok := e.Object.(*eventloggerv1.EventLogger); ok {
		return p.Config.matches(e.Object)
	}
	return p.logEvent(e.Object)
}

// Update implements Predicate
func (p *loggingPredicate) Update(e event.UpdateEvent) bool {
	if _, ok := e.ObjectNew.(*eventloggerv1.EventLogger); ok {
		return p.Config.matches(e.ObjectNew)
	}
	return p.logEvent(e.ObjectNew)
}

// Delete implements Predicate
func (p *loggingPredicate) Delete(e event.DeleteEvent) bool {
	if _, ok := e.Object.(*eventloggerv1.EventLogger); ok {
		return p.Config.matches(e.Object)
	}
	return false
}

func (p *loggingPredicate) logEvent(e runtime.Object) bool {
	if p.Config == nil || p.Config.filter == nil {
		return false
	}

	evt, ok := e.(*corev1.Event)
	if !ok {
		return false
	}
	if evt.ResourceVersion <= p.lastVersion {
		return false
	}
	p.lastVersion = evt.ResourceVersion // SA4005:

	if p.Config.filter.Match(evt) {
		var eventLogger logr.Logger
		if len(p.Config.logFields) == 0 {
			ts := evt.LastTimestamp
			if ts.IsZero() {
				ts = evt.FirstTimestamp
			}
			if ts.IsZero() {
				ts = metav1.Time{Time: evt.EventTime.Time}
			}

			eventLogger = eventLog.WithValues(
				"namespace", evt.Namespace,
				"name", evt.Name,
				"reason", evt.Reason,
				"timestamp", ts,
				"type", evt.Type,
				"involvedObject", evt.InvolvedObject,
				"source", evt.Source,
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
				} else if lf.Value != nil {
					eventLogger = eventLogger.WithValues(lf.Name, *lf.Value)
				}
			}
		}

		eventLogger.Info(evt.Message)
	}
	return false
}

func getLatestRevision(ctx context.Context, cl client.Client, namespace string) (string, error) {
	eventList := &corev1.EventList{}
	opts := []client.ListOption{
		client.Limit(0),
		client.InNamespace(namespace),
	}

	err := cl.List(ctx, eventList, opts...)
	if err != nil {
		return "", err
	}
	return eventList.ResourceVersion, nil
}

// SetupWithManager setup with manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, namespace string) error {
	cl, err := client.New(mgr.GetConfig(), client.Options{})
	if err != nil {
		return err
	}

	lv, err := getLatestRevision(context.Background(), cl, namespace)
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventloggerv1.EventLogger{}).
		Watches(&corev1.Event{}, &handler.Funcs{}).
		WithEventFilter(&loggingPredicate{Config: r.Config, lastVersion: lv}).
		Complete(r)
}
