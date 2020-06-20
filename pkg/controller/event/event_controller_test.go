package event

import (
	"encoding/json"
	"regexp"
	"testing"

	v1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	"github.com/bakito/k8s-event-logger-operator/pkg/mock/logr"
	"github.com/golang/mock/gomock"
	. "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
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
	Config   v1.EventLoggerSpec `json:"config"`
	Event    corev1.Event       `json:"event"`
	Expected bool               `json:"expected"`
}{
	{
		v1.EventLoggerSpec{},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Warning"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "ConfigMap"}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Normal"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Warning"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}}, EventTypes: []string{"Warning"}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varFalse}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		false,
	},
	{
		v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: &varTrue}}},
		corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
		true,
	},
}

func Test_shouldLog(t *testing.T) {
	for i, data := range shouldLogData {
		lp := &loggingPredicate{cfg: &config{filter: newFilter(data.Config)}}

		dStr, err := json.Marshal(&shouldLogData[i])
		Assert(t, is.Nil(err))

		Assert(t, lp.shouldLog(&data.Event) == data.Expected, "ShouldLogData #%v: %s", i, string(dStr))
	}
}

func Test_logEvent_no_filter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := logr.NewMockLogger(ctrl)
	eventLog = mock
	mock.EXPECT().WithValues().Times(0)

	lp := &loggingPredicate{}
	lp.logEvent(&metav1.ObjectMeta{Namespace: testNamespace}, &corev1.Event{})
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

	lp.logEvent(&metav1.ObjectMeta{Namespace: testNamespace}, &corev1.Event{
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
		cfg:         &config{filter: &Filter{}},
	}

	lp.logEvent(&metav1.ObjectMeta{Namespace: testNamespace}, &corev1.Event{
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
	parent.EXPECT().WithValues(gomock.Any(), gomock.Any()).Times(1).Return(child)
	child.EXPECT().WithValues(gomock.Any(), gomock.Any()).Times(2).Return(child)
	child.EXPECT().Info(gomock.Any()).Times(1)

	lp := &loggingPredicate{
		lastVersion: "2",
		cfg: &config{filter: &Filter{},
			logFields: []v1.LogField{
				{Name: "type", Path: []string{"Type"}},
				{Name: "name", Path: []string{"ObjectMeta", "Name"}},
				{Name: "kind", Path: []string{"InvolvedObject", "Kind"}},
			},
		},
	}

	lp.logEvent(&metav1.ObjectMeta{Namespace: testNamespace}, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{ResourceVersion: "3"}, Type: "test-type",
		InvolvedObject: corev1.ObjectReference{Kind: "test-kind"},
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
	cfg := &config{}
	r := newReconciler(cl, s, cfg)

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

	r := newReconciler(cl, s, &config{})

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
