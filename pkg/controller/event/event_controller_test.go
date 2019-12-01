package event

import (
	"testing"

	v1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	. "gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_matches(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.matches([]string{"abc", "xyz"}, "abc"))
	Assert(t, lp.matches([]string{"abc", "xyz"}, "^abc$"))
	Assert(t, !lp.matches([]string{"abc", "xyz"}, "^ab$"))
}

func Test_contains(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.contains([]string{"abc", "xyz"}, "abc"))
	Assert(t, lp.contains([]string{"abc", "xyz"}, "xyz"))
	Assert(t, !lp.contains([]string{"abc", "xyz"}, "xxx"))
}

var shouldLogData = []struct {
	config   v1.EventLoggerSpec
	event    corev1.Event
	expected bool
}{
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "ConfigMap"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}, EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Normal"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Warning"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Normal"}}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*Message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
}

func Test_shouldLog(t *testing.T) {
	for i, data := range shouldLogData {
		lp := &loggingPredicate{}
		lp.init(&data.config)

		Assert(t, lp.shouldLog(&data.event) == data.expected, "ShouldLogData #%v: %v", i, data)
	}
}
