package event

import (
	"regexp"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_event")

// Add creates a new Event Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config *eventloggerv1.EventLoggerSpec) error {
	// Create a new controller
	c, err := controller.New("event-controller", mgr, controller.Options{Reconciler: reconcile.Func(nil)})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Event
	p := &loggingPredicate{}
	p.init(config)
	err = c.Watch(&source.Kind{Type: &corev1.Event{}}, &handler.Funcs{}, p)
	if err != nil {
		return err
	}

	return nil
}

type loggingPredicate struct {
	predicate.Funcs
	lastVersion string

	kinds map[string]*eventloggerv1.Kind
}

func (p *loggingPredicate) init(config *eventloggerv1.EventLoggerSpec) {
	// TODO pre init regex pattern
	p.kinds = make(map[string]*eventloggerv1.Kind)
	for _, k := range config.Kinds {
		kp := &k
		p.kinds[k.Name] = kp
		if kp.EventTypes == nil {
			kp.EventTypes = config.EventTypes
		}
	}
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
	evt := e.(*corev1.Event)
	if p.shouldLog(evt) {
		eventLogger := log.WithValues(
			"Namespace", mo.GetNamespace(),
			"Name", mo.GetName(),
			"Reason", evt.Reason,
			"Timestamp", evt.LastTimestamp,
			"Type", evt.Type,
			"InvolvedObject.Kind ", evt.InvolvedObject.Kind,
			"InvolvedObject.Namespace ", evt.InvolvedObject.Namespace,
			"InvolvedObject.Name ", evt.InvolvedObject.Name,
			"ResourceVersion ", evt.ResourceVersion,
			"ReportingController ", evt.ReportingController,
			"Source ", evt.Source,
		)
		eventLogger.Info(evt.Message)
	}
	return false
}

func (p *loggingPredicate) shouldLog(e *corev1.Event) bool {
	k, ok := p.kinds[e.InvolvedObject.Kind]
	if !ok {
		return false
	}

	if !p.contains(k.EventTypes, e.Type) {
		return false
	}

	return p.matches(k.MatchingPatterns, e.Message)
}

func (p *loggingPredicate) matches(list []string, val string) bool {
	if len(list) == 0 {
		return true
	}
	for _, v := range list {
		var p = regexp.MustCompile(v)
		if p.MatchString(val) {
			return true
		}
	}
	return false
}

func (p *loggingPredicate) contains(list []string, val string) bool {
	if len(list) == 0 {
		return true
	}
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
