package logging

import (
	"context"
	"encoding/json"
	"time"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/pkg/filter"
	mc "github.com/bakito/k8s-event-logger-operator/pkg/mocks/client"
	ml "github.com/bakito/k8s-event-logger-operator/pkg/mocks/logr"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "eventlogger-operator"
	testName      = "eventlogger-operator-name"
)

var _ = Describe("Logging", func() {
	var (
		mockCtrl *gm.Controller
		ctx      context.Context
		cl       *mc.MockClient
	)

	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		cl = mc.NewMockClient(mockCtrl)
		ctx = context.Background()
	})
	AfterEach(func() {
		defer mockCtrl.Finish()
	})

	Context("Reconcile", func() {
		var (
			s   *runtime.Scheme
			r   *Reconciler
			req reconcile.Request
		)

		BeforeEach(func() {
			s = scheme.Scheme
			Ω(v1.SchemeBuilder.AddToScheme(s)).ShouldNot(HaveOccurred())
			r = &Reconciler{
				Client:     cl,
				Log:        ctrl.Log.WithName("controllers").WithName("Event"),
				Scheme:     s,
				Config:     &Config{},
				LoggerMode: false,
			}
			req = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "foo",
					Namespace: testNamespace,
				},
			}
		})

		Context("Update", func() {
			It("should update an existing if LoggerMode is disabled", func() {
				r.LoggerMode = false
				cl.EXPECT().Get(gm.Any(), gm.Any(), gm.Any())
				cl.EXPECT().Update(gm.Any(), gm.Any(), gm.Any())
				_, err := r.Reconcile(ctx, req)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r.Config.filter).ShouldNot(BeNil())
			})
			It("should not update an existing if LoggerMode is enabled", func() {
				r.LoggerMode = true
				cl.EXPECT().Get(gm.Any(), gm.Any(), gm.Any())
				_, err := r.Reconcile(ctx, req)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r.Config.filter).ShouldNot(BeNil())
			})
		})

		It("should do noting if not found", func() {
			cl.EXPECT().
				Get(gm.Any(), gm.Any(), gm.Any()).
				Return(errors.NewNotFound(v1.GroupVersion.WithResource("").GroupResource(), ""))
			_, err := r.Reconcile(ctx, req)
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("logEvent", func() {
		var mockSink *ml.MockLogSink

		BeforeEach(func() {
			mockSink = ml.NewMockLogSink(mockCtrl)
			mockSink.EXPECT().Init(gm.Any())
			mockSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
			eventLog = logr.New(mockSink)
		})

		It("should log nothing", func() {
			mockSink.EXPECT().WithValues().Times(0)

			lp := &loggingPredicate{}
			lp.logEvent(&corev1.Event{})
		})
		It("should log nothing if object is not an event", func() {
			mockSink.EXPECT().WithValues().Times(0)

			lp := &loggingPredicate{
				lastVersion: "2",
				Config:      &Config{filter: filter.Always},
			}

			lp.logEvent(&corev1.Pod{})
		})
		It("should log nothing if resource version does not match", func() {
			mockSink.EXPECT().WithValues().Times(0)

			lp := &loggingPredicate{
				lastVersion: "2",
				Config:      &Config{filter: filter.Always},
			}

			lp.logEvent(&corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
			})
		})
		It("should log one message with 14 fields", func() {
			childSink := ml.NewMockLogSink(mockCtrl)
			childSink.EXPECT().Init(gm.Any()).AnyTimes()
			childSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
			mockSink.EXPECT().WithValues(repeat(gm.Any(), 14)...).Times(1).Return(childSink)
			childSink.EXPECT().Info(gm.Any(), gm.Any()).Times(1)

			lp := &loggingPredicate{
				lastVersion: "2",
				Config:      &Config{filter: filter.Always},
			}

			lp.logEvent(&corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "3",
				},
			})
		})
		It("should log one message with custom fields", func() {
			childSink := ml.NewMockLogSink(mockCtrl)
			childSink.EXPECT().Init(gm.Any()).AnyTimes()
			childSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
			mockSink.EXPECT().WithValues("type", "test-type").Times(1).Return(childSink)
			childSink.EXPECT().WithValues("name", "test-io-name").Times(1).Return(childSink)
			childSink.EXPECT().WithValues("kind", "test-kind").Times(1).Return(childSink)
			childSink.EXPECT().WithValues("reason", "").Times(1).Return(childSink)
			childSink.EXPECT().Info(gm.Any(), gm.Any()).Times(1)

			lp := &loggingPredicate{
				Config: &Config{
					filter: filter.Always,
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
		})
		It("should resolve timestamp", func() {
			childSink := ml.NewMockLogSink(mockCtrl)
			childSink.EXPECT().Init(gm.Any()).AnyTimes()
			childSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
			mockSink.EXPECT().WithValues(repeat(gm.Any(), 14)...).Times(3).DoAndReturn(
				func(a ...interface{}) interface{} {
					t, ok := a[7].(metav1.Time)
					if !ok || t.IsZero() {
						Fail("timestamp not set")
					}
					return childSink
				})
			childSink.EXPECT().Info(gm.Any(), gm.Any()).Times(3)

			lp := &loggingPredicate{
				Config: &Config{filter: filter.Always},
			}

			lp.logEvent(&corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "3",
				},
				LastTimestamp: metav1.Now(),
			})
			lp.logEvent(&corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "4",
				},
				FirstTimestamp: metav1.Now(),
			})
			lp.logEvent(&corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "5",
				},
				EventTime: metav1.MicroTime{Time: time.Now()},
			})
		})

		DescribeTable("the > inequality",
			func(config v1.EventLoggerSpec, event corev1.Event, expected bool, description string) {
				data := &sld{config, event, expected, description}
				lp := &loggingPredicate{Config: &Config{filter: newFilter(data.Config)}}

				_, err := json.Marshal(&data)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lp.Config.filter.Match(&data.Event)).Should(Equal(expected))
				Ω(lp.Config.filter.String()).Should(Equal(data.Description))
			},
			Entry("1",
				v1.EventLoggerSpec{},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
				true,
				"true",
			),
			Entry("2",
				v1.EventLoggerSpec{EventTypes: []string{}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				true,
				"true",
			),
			Entry("3",
				v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				true,
				"( EventType in [Normal] )",
			),
			Entry("4",
				v1.EventLoggerSpec{EventTypes: []string{"Normal"}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Warning"},
				false,
				"( EventType in [Normal] )",
			),
			Entry("5",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
				true,
				"( ( ( Kind == 'Pod' ) ) )",
			),
			Entry("6",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "ConfigMap"}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
				false,
				"( ( ( Kind == 'ConfigMap' ) ) )",
			),
			Entry("7",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}},
				true,
				"( ( ( Kind == 'Pod' ) ) )",
			),
			Entry(
				"8",

				v1.EventLoggerSpec{
					Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}, Reasons: []string{"Created", "Started"}}},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Created"},
				true,
				"( ( ( Kind == 'Pod' AND Reason in [Created, Started] ) ) )",
			),
			Entry(
				"9",

				v1.EventLoggerSpec{
					Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{}, Reasons: []string{"Created", "Started"}}},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Killing"},
				false,
				"( ( ( Kind == 'Pod' AND Reason in [Created, Started] ) ) )",
			),
			Entry(
				"10",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{{Name: "Application", APIGroup: ptr.To("argoproj.io"), EventTypes: []string{}}},
				},
				corev1.Event{
					InvolvedObject: corev1.ObjectReference{
						Kind:       "Application",
						APIVersion: schema.GroupVersion{Group: "argoproj.io", Version: "v1alpha1"}.String(),
					},
				},
				true,
				"( ( ( Kind == 'Application' AND APIGroup == 'argoproj.io' ) ) )",
			),
			Entry(
				"11",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{{Name: "Application", APIGroup: ptr.To("argoproj.io"), EventTypes: []string{}}},
				},
				corev1.Event{
					InvolvedObject: corev1.ObjectReference{
						Kind:       "Application",
						APIVersion: schema.GroupVersion{Group: "app.k8s.io", Version: "v1beta1"}.String(),
					},
				},
				false,
				"( ( ( Kind == 'Application' AND APIGroup == 'argoproj.io' ) ) )",
			),
			Entry("12",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Normal"}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				true,
				"( EventType in [Normal] OR ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
			),
			Entry("13",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod"}}, EventTypes: []string{"Warning"}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				false,
				"( EventType in [Warning] OR ( ( Kind == 'Pod' AND EventType in [Warning] ) ) )",
			),
			Entry("14",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				true,
				"( ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
			),
			Entry("15",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", EventTypes: []string{"Warning"}}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				false,
				"( ( ( Kind == 'Pod' AND EventType in [Warning] ) ) )",
			),
			Entry(
				"16",

				v1.EventLoggerSpec{
					Kinds:      []v1.Kind{{Name: "Pod", EventTypes: []string{"Normal"}}},
					EventTypes: []string{"Warning"},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Type: "Normal"},
				true,
				"( EventType in [Warning] OR ( ( Kind == 'Pod' AND EventType in [Normal] ) ) )",
			),
			Entry("17",
				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*message.*"}}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				true,
				"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*message.*/ ) ) ) ) )",
			),
			Entry("18",
				v1.EventLoggerSpec{Kinds: []v1.Kind{{Name: "Pod", MatchingPatterns: []string{".*Message.*"}}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				false,
				"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*Message.*/ ) ) ) ) )",
			),
			Entry(
				"19",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: ptr.To(false)},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				true,
				"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*message.*/ ) ) ) ) )",
			),
			Entry(
				"20",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: ptr.To(false)},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				false,
				"( ( ( Kind == 'Pod' AND ( false XOR ( Message matches /.*Message.*/ ) ) ) ) )",
			),
			Entry(
				"21",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", MatchingPatterns: []string{".*message.*"}, SkipOnMatch: ptr.To(true)},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				false,
				"( ( ( Kind == 'Pod' AND ( true XOR ( Message matches /.*message.*/ ) ) ) ) )",
			),
			Entry(
				"22",
				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", MatchingPatterns: []string{".*Message.*"}, SkipOnMatch: ptr.To(true)},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Message: "This is a test message"},
				true,
				"( ( ( Kind == 'Pod' AND ( true XOR ( Message matches /.*Message.*/ ) ) ) ) )",
			),
			Entry(
				"23",

				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", EventTypes: []string{}, SkipReasons: []string{"Created", "Started"}},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Created"},
				false,
				"( ( ( Kind == 'Pod' AND Reason NOT in [Created, Started] ) ) )",
			),
			Entry(
				"24",

				v1.EventLoggerSpec{
					Kinds: []v1.Kind{
						{Name: "Pod", EventTypes: []string{}, SkipReasons: []string{"Created", "Started"}},
					},
				},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Killing"},
				true,
				"( ( ( Kind == 'Pod' AND Reason NOT in [Created, Started] ) ) )",
			),
			Entry("25",

				v1.EventLoggerSpec{Kinds: []v1.Kind{{
					Name: "Pod", EventTypes: []string{},
					SkipReasons: []string{"Created"},
					Reasons:     []string{"Created"},
				}}},
				corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: "Pod"}, Reason: "Created"},
				false,
				"( ( ( Kind == 'Pod' AND Reason NOT in [Created] AND Reason in [Created] ) ) )",
			),
		)
	})

	Context("contains", func() {
		It("should contain the value", func() {
			Ω(contains([]string{"abc", "xyz"}, "abc")).Should(BeTrue())
			Ω(contains([]string{"abc", "xyz"}, "xyz")).Should(BeTrue())
		})
		It("should not contain the value", func() {
			Ω(contains([]string{"abc", "xyz"}, "xxx")).Should(BeFalse())
		})
	})

	Context("loggingPredicate", func() {
		var (
			lp *loggingPredicate
			el *v1.EventLogger
		)
		BeforeEach(func() {
			lp = &loggingPredicate{
				Config: &Config{
					watchNamespace: testNamespace,
					podNamespace:   "",
					name:           testName,
				},
			}
			el = &v1.EventLogger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testName,
				},
			}
		})
		Context("Create", func() {
			It("should match for reconciling with watchNamespace", func() {
				Ω(lp.Create(event.CreateEvent{Object: el})).Should(BeTrue())
			})
			It("should not match for reconciling with watchNamespace", func() {
				el.Name = "foo"
				Ω(lp.Create(event.CreateEvent{Object: el})).Should(BeFalse())
			})
			It("should match for reconciling with podNamespace", func() {
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Create(event.CreateEvent{Object: el})).Should(BeTrue())
			})
			It("should match for reconciling with podNamespace", func() {
				el.Name = "foo"
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Create(event.CreateEvent{Object: el})).Should(BeFalse())
			})
			It("should not reconcile or log for another object", func() {
				pod := &corev1.Pod{}
				Ω(lp.Create(event.CreateEvent{Object: pod})).Should(BeFalse())
			})
		})
		Context("Update", func() {
			It("should match for reconciling with watchNamespace", func() {
				Ω(lp.Update(event.UpdateEvent{ObjectNew: el})).Should(BeTrue())
			})
			It("should not match for reconciling with watchNamespace", func() {
				el.Name = "foo"
				Ω(lp.Update(event.UpdateEvent{ObjectNew: el})).Should(BeFalse())
			})
			It("should match for reconciling with podNamespace", func() {
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Update(event.UpdateEvent{ObjectNew: el})).Should(BeTrue())
			})
			It("should match for reconciling with podNamespace", func() {
				el.Name = "foo"
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Update(event.UpdateEvent{ObjectNew: el})).Should(BeFalse())
			})
			It("should not reconcile or log for another object", func() {
				pod := &corev1.Pod{}
				Ω(lp.Update(event.UpdateEvent{ObjectNew: pod})).Should(BeFalse())
			})
		})
		Context("Delete", func() {
			It("should match for reconciling with watchNamespace", func() {
				Ω(lp.Delete(event.DeleteEvent{Object: el})).Should(BeTrue())
			})
			It("should not match for reconciling with watchNamespace", func() {
				el.Name = "foo"
				Ω(lp.Delete(event.DeleteEvent{Object: el})).Should(BeFalse())
			})
			It("should match for reconciling with podNamespace", func() {
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Delete(event.DeleteEvent{Object: el})).Should(BeTrue())
			})
			It("should match for reconciling with podNamespace", func() {
				el.Name = "foo"
				lp.Config.watchNamespace = ""
				lp.Config.podNamespace = testNamespace
				Ω(lp.Delete(event.DeleteEvent{Object: el})).Should(BeFalse())
			})
			It("should not reconcile or log for another object", func() {
				pod := &corev1.Pod{}
				Ω(lp.Delete(event.DeleteEvent{Object: pod})).Should(BeFalse())
			})
		})
	})
	Context("getLatestRevision", func() {
		It("should find the last revision", func() {
			cl.EXPECT().
				List(gm.Any(), gm.Any(), gm.Any()).
				Do(func(_ context.Context, el *corev1.EventList, _ ...client.ListOption) {
					el.ResourceVersion = "3"
				})
			rev, err := getLatestRevision(ctx, cl, "")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(rev).Should(Equal("3"))
		})
	})
})

type sld struct {
	Config      v1.EventLoggerSpec `json:"config"`
	Event       corev1.Event       `json:"event"`
	Expected    bool               `json:"expected"`
	Description string             `json:"description"`
}

func repeat(m gm.Matcher, times int) []interface{} {
	var list []interface{}
	for i := 0; i < times; i++ {
		list = append(list, m)
	}
	return list
}
