package event

import (
	"regexp"
	"testing"

	"encoding/json"

	v1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	. "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
)

var (
	varTrue  = true
	varFalse = false
)

func Test_matches(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.matches([]*regexp.Regexp{regexp.MustCompile("abc")}, false, "abc"))
	Assert(t, lp.matches([]*regexp.Regexp{regexp.MustCompile("^abc$")}, false, "abc"))
	Assert(t, !lp.matches([]*regexp.Regexp{regexp.MustCompile("^ab$")}, false, "abc"))
}

func Test_contains(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.contains([]string{"abc", "xyz"}, "abc"))
	Assert(t, lp.contains([]string{"abc", "xyz"}, "xyz"))
	Assert(t, !lp.contains([]string{"abc", "xyz"}, "xxx"))
}

var shouldLogData = []struct {
	Config   v1.EventLoggerConf `json:"config"`
	Event    corev1.Event       `json:"event"`
	Expected bool               `json:"expected"`
}{
	{
		v1.EventLoggerConf{},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerConf{EventTypes: []string{}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerConf{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerConf{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Warning"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "ConfigMap"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}, EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod"}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Normal"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Warning"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", EventTypes: []string{"Normal"}}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*Message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerConf{Kinds: []v1.Kind{v1.Kind{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
}

func Test_shouldLog(t *testing.T) {
	for i, data := range shouldLogData {
		lp := &loggingPredicate{}
		lp.init(&data.Config)

		dStr, err := json.Marshal(&shouldLogData[i])
		Assert(t, is.Nil(err))

		Assert(t, lp.shouldLog(&data.Event) == data.Expected, "ShouldLogData #%v: %s", i, string(dStr))
	}
}
