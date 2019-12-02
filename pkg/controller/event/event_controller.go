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
var eventLog = logf.Log.WithName("event")

// Add creates a new Event Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config *eventloggerv1.EventLoggerConf) error {
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

	kinds map[string]*filter
}

func (p *loggingPredicate) init(config *eventloggerv1.EventLoggerConf) {
	p.kinds = make(map[string]*filter)
	for _, k := range config.Kinds {
		kp := &k
		p.kinds[k.Name] = &filter{
			matchingPatterns: []*regexp.Regexp{},
		}
		if kp.EventTypes == nil {
			p.kinds[k.Name].eventTypes = config.EventTypes
		} else {
			p.kinds[k.Name].eventTypes = kp.EventTypes
		}

		if k.MatchingPatterns != nil {
			if k.SkipOnMatch != nil && *k.SkipOnMatch {
				p.kinds[k.Name].skipOnMatch = true
			}
			for _, mp := range k.MatchingPatterns {

				p.kinds[k.Name].matchingPatterns = append(p.kinds[k.Name].matchingPatterns, regexp.MustCompile(mp))
			}
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
	f, ok := p.kinds[e.InvolvedObject.Kind]
	if !ok {
		return false
	}

	if !p.contains(f.eventTypes, e.Type) {
		return false
	}

	return p.matches(f.matchingPatterns, f.skipOnMatch, e.Message)
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

type filter struct {
	eventTypes       []string
	matchingPatterns []*regexp.Regexp
	skipOnMatch      bool
}
