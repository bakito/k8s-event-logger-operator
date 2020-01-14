package event

import (
	"context"
	"fmt"
	"regexp"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	filter   *Filter
)

// Add creates a new Event Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, name string) error {
	return add(mgr, newReconciler(mgr.GetClient(), mgr.GetScheme(), name))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(client client.Client, scheme *runtime.Scheme, name string) reconcile.Reconciler {
	return &ReconcileConfig{client: client, scheme: scheme, name: name}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("event-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EventLogger
	err = c.Watch(&source.Kind{Type: &eventloggerv1.EventLogger{}}, &handler.EnqueueRequestForObject{})

	// Watch for changes to primary resource Event
	p := &loggingPredicate{}
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
	client  client.Client
	scheme  *runtime.Scheme
	name    string
	ignored map[string]bool
}

// Reconcile reads that state of the cluster for a EventLogger object and makes changes based on the state read
// and what is in the EventLogger.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "CR.Name", r.name)

	if r.name != "" && r.name != request.Name {
		if _, ok := r.ignored[request.Name]; ok {
			reqLogger.V(4).Info("ignore this event logger config")
		} else {
			reqLogger.Error(fmt.Errorf(""), "ignore this event logger config due to a different event logger config in the same namespace")
		}
		return reconcile.Result{}, nil
	}

	if r.name == "" {
		r.name = request.Name
		reqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "CR.Name", r.name)
	}

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			filter = nil
			reqLogger.Info("cr was deleted, removing filter")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return r.updateCR(cr, reqLogger, err)
	}

	newFilter := newFilter(cr.Spec)
	if filter == nil || !filter.Equals(newFilter) {
		filter = newFilter
		reqLogger.WithValues("filter", filter).Info("apply new filter")
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
	if filter == nil {
		return false
	}

	evt := e.(*corev1.Event)
	if evt.ResourceVersion <= p.lastVersion {
		return false
	}
	p.lastVersion = evt.ResourceVersion

	if p.shouldLog(evt) {
		eventLogger := eventLog.WithValues(
			"namespace", mo.GetNamespace(),
			"name", mo.GetName(),
			"reason", evt.Reason,
			"timestamp", evt.LastTimestamp,
			"type", evt.Type,
			"involvedObject ", evt.InvolvedObject,
			"source ", evt.Source,
		)
		eventLogger.Info(evt.Message)
	}
	return false
}

func (p *loggingPredicate) shouldLog(e *corev1.Event) bool {

	if len(filter.Kinds) == 0 {
		return len(filter.EventTypes) == 0 || p.contains(filter.EventTypes, e.Type)
	}

	k, ok := filter.Kinds[e.InvolvedObject.Kind]
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
