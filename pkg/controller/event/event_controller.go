package event

import (
	"context"
	"reflect"
	"regexp"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	"github.com/fatih/structs"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log      = logf.Log.WithName("controller_event")
	eventLog = logf.Log.WithName("event")
)

// Add creates a new Event Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, namespace string, name string) error {
	cfg := &config{
		namespace: namespace,
		name:      name,
	}
	return add(mgr, newReconciler(mgr.GetClient(), mgr.GetScheme(), cfg), cfg)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(client client.Client, scheme *runtime.Scheme, cfg *config) reconcile.Reconciler {
	return &ReconcileConfig{client: client, scheme: scheme, cfg: cfg}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config) error {
	// Create a new controller
	c, err := controller.New("event-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EventLogger
	err = c.Watch(&source.Kind{Type: &eventloggerv1.EventLogger{}}, &handler.EnqueueRequestForObject{}, eventLoggerPredicate{cfg: cfg})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Event
	p := &loggingPredicate{cfg: cfg}
	p.lastVersion, err = getLatestRevision(mgr)
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &corev1.Event{}}, &handler.Funcs{}, p)
}

// blank assignment to verify that ReconcileConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileConfig{}

// ReconcileConfig reconciles a EventLogger object
type ReconcileConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	cfg    *config
}

// Reconcile reads that state of the cluster for a EventLogger object and makes changes based on the state read
// and what is in the EventLogger.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	if r.cfg.name == "" {
		r.cfg.name = request.Name
	}
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "CR.Name", r.cfg.name)

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cr)
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

func (r *ReconcileConfig) updateCR(cr *eventloggerv1.EventLogger, logger logr.Logger, err error) (reconcile.Result, error) {
	updErr := cr.UpdateStatus(logger, err, r.client)
	return reconcile.Result{}, updErr
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
