package logging

import (
	"encoding/json"
	"testing"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/pkg/filter"
	"github.com/bakito/k8s-event-logger-operator/pkg/mock/logr"
	"github.com/golang/mock/gomock"
	. "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "eventlogger-operator"
)

var (
	varTrue  = true
	varFalse = false
)

func Test_contains(t *testing.T) {
	Assert(t, contains([]string{"abc", "xyz"}, "abc"))
	Assert(t, contains([]string{"abc", "xyz"}, "xyz"))
	Assert(t, !contains([]string{"abc", "xyz"}, "xxx"))
}

var shouldLogData = []struct {
	Config      v1.EventLoggerSpec `json:"config"`
	Event       corev1.Event       `json:"event"`
	Expected    bool               `json:"expected"`
	Description string             `json:"description"`
}{
	{
		v1.EventLoggerSpec{},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
		"true",
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
		"true",
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
		"( EventType in [Normal] )",
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Warning"},
		false,
		"( EventType in [Normal] )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
		"( ( ( Kind == 'Pod' ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "ConfigMap"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		false,
		"( ( ( Kind == 'ConfigMap' ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
		"( ( ( Kind == 'Pod' ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}, Reasons: []string{"Created", "Started"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Created"},
		true,
		"( ( ( Kind == 'Pod' AND Reason in [Created, Started] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}, Reasons: []string{"Created", "Started"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Killing"},
		false,
		"( ( ( Kind == 'Pod' AND Reason in [Created, Started] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Application", ApiGroup: "argoproj.io", EventTypes: []string{}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Application", APIVersion: schema.GroupVersion{Group: "argoproj.io", Version: "v1alpha1"}.String()}},
		true,
		"( ( ( Kind == 'Application' AND ApiGroup == 'argoproj.io' ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Application", ApiGroup: "argoproj.io", EventTypes: []string{}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Application", APIVersion: schema.GroupVersion{Group: "app.k8s.io", Version: "v1beta1"}.String()}},
		false,
		"( ( ( Kind == 'Application' AND ApiGroup == 'argoproj.io' ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
		"( EventType in [Normal] OR ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
		"( EventType in [Warning] OR ( ( Kind == 'Pod' AND EventType in [Warning] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
		"( ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Warning"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
		"( ( ( Kind == 'Pod' AND EventType in [Warning] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
		"( EventType in [Warning] OR ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
		"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*message.*/ ) ) ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
		"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*Message.*/ ) ) ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
		"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*message.*/ ) ) ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
		"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*Message.*/ ) ) ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
		"( ( ( Kind == 'Pod' AND ( true XOR ( Message matches /.*message.*/ ) ) ) ) )",
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
		"( ( ( Kind == 'Pod' AND ( true XOR ( Message matches /.*Message.*/ ) ) ) ) )",
	},
}

func Test_shouldLog(t *testing.T) {
	for i, data := range shouldLogData {
		lp := &loggingPredicate{Config: &Config{filter: newFilter(data.Config)}}

		dStr, err := json.Marshal(&shouldLogData[i])
		Assert(t, is.Nil(err))

		Assert(t, lp.Config.filter.Match(&data.Event) == data.Expected, "ShouldLogData #%v: %s", i, string(dStr))
		Assert(t, lp.Config.filter.String() == data.Description)
	}
}

func Test_logEvent_no_filter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := logr.NewMockLogger(ctrl)
	eventLog = mock
	mock.EXPECT().WithValues().Times(0)

	lp := &loggingPredicate{}
	lp.logEvent(&corev1.Event{})
}

func Test_logEvent_wrong_resource_version(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := logr.NewMockLogger(ctrl)
	eventLog = mock
	mock.EXPECT().WithValues().Times(0)

	lp := &loggingPredicate{
		lastVersion: "2",
	}

	lp.logEvent(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "1",
		},
	})
}

func Test_logEvent_true(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := logr.NewMockLogger(ctrl)
	child := logr.NewMockLogger(ctrl)
	eventLog = parent
	parent.EXPECT().WithValues(repeat(gomock.Any(), 14)...).Times(1).Return(child)
	child.EXPECT().Info(gomock.Any()).Times(1)

	lp := &loggingPredicate{
		lastVersion: "2",
		Config:      &Config{filter: filter.Always},
	}

	lp.logEvent(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "3",
		},
	})
}

func Test_logEvent_true_custom_fields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := logr.NewMockLogger(ctrl)
	child := logr.NewMockLogger(ctrl)
	eventLog = parent
	parent.EXPECT().WithValues("type", "test-type").Times(1).Return(child)
	child.EXPECT().WithValues("name", "test-io-name").Times(1).Return(child)
	child.EXPECT().WithValues("kind", "test-kind").Times(1).Return(child)
	child.EXPECT().WithValues("reason", "").Times(1).Return(child)
	child.EXPECT().Info(gomock.Any()).Times(1)

	lp := &loggingPredicate{
		Config: &Config{filter: filter.Always,
			logFields: []v1.LogField{
				{Name: "type", Path: []string{"Type"}},
				{Name: "name", Path: []string{"InvolvedObject", "Name"}},
				{Name: "kind", Path: []string{"InvolvedObject", "Kind"}},
				{Name: "reason", Path: []string{"Reason"}},
			},
		},
	}

	lp.logEvent(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "3",
			Name:            "test-event-name",
		},
		Type: "test-type",
		InvolvedObject: corev1.ObjectReference{
			Kind: "test-kind",
			Name: "test-io-name",
		},
		Reason: "",
	})
}

func Test_Reconcile_existing(t *testing.T) {
	s := scheme.Scheme
	Assert(t, is.Nil(v1.SchemeBuilder.AddToScheme(s)))
	el := &v1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: v1.EventLoggerSpec{
			Kinds: []v1.Kind{
				{
					Name: "Pod",
				},
			},
		},
	}

	cl := fake.NewFakeClientWithScheme(s, el)
	cfg := &Config{}
	r := &Reconciler{
		Client: cl,
		Log:    ctrl.Log.WithName("controllers").WithName("Event"),
		Scheme: s,
		Config: cfg,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
	}
	_, err := r.Reconcile(req)
	Assert(t, is.Nil(err))
	Assert(t, cfg.filter != nil)

}

func Test_Reconcile_deleted(t *testing.T) {

	s := scheme.Scheme
	Assert(t, is.Nil(v1.SchemeBuilder.AddToScheme(s)))

	cl := fake.NewFakeClientWithScheme(s)

	r := &Reconciler{
		Client: cl,
		Log:    ctrl.Log.WithName("controllers").WithName("Event"),
		Scheme: s,
		Config: &Config{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "foo",
			Namespace: testNamespace,
		},
	}
	_, err := r.Reconcile(req)
	Assert(t, is.Nil(err))
}

func repeat(m gomock.Matcher, times int) []interface{} {
	var list []interface{}
	for i := 0; i < times; i++ {
		list = append(list, m)
	}
	return list
}
