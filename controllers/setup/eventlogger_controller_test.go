package setup

import (
	"context"
	"reflect"
	"time"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	c "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/version"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "eventlogger-operator"
	testImage     = "quay.io/bakito/k8s-event-logger"
)

var _ = Describe("Logging", func() {
	var (
		mockCtrl *gm.Controller
		ns2      string
		el       *v1.EventLogger
	)

	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		defaultPodReqCPU = resource.MustParse("111m")
		defaultPodReqMem = resource.MustParse("222Mi")
		defaultPodMaxCPU = resource.MustParse("333m")
		defaultPodMaxMem = resource.MustParse("444Mi")
		ns2 = "eventlogger-operators"

		el = &v1.EventLogger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "eventlogger",
				Namespace: testNamespace,
			},
			Spec: v1.EventLoggerSpec{
				Labels:        map[string]string{"test-label": "foo"},
				Annotations:   map[string]string{"test-annotation": "bar"},
				ScrapeMetrics: pointer.BoolPtr(true),
				Namespace:     &ns2,
				NodeSelector:  map[string]string{"ns-key": "ns-value"},
			},
			Status: v1.EventLoggerStatus{
				OperatorVersion: "0",
				Hash:            "",
				LastProcessed:   metav1.Date(2020, 1, 1, 1, 1, 1, 1, time.Local),
			},
		}
	})
	AfterEach(func() {
		defer mockCtrl.Finish()
	})

	Context("Reconcile", func() {
		Context("EventLogger", func() {
			It("update the eventlogger", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// check updated status
				updated := &v1.EventLogger{}
				err := cl.Get(context.TODO(), types.NamespacedName{
					Name:      "eventlogger",
					Namespace: testNamespace,
				}, updated)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(updated.Status.LastProcessed.String()).ShouldNot(BeEmpty())
				Ω(updated.Status.Hash).ShouldNot(BeEmpty())
				Ω(updated.Status.OperatorVersion).Should(Equal(version.Version))
			})
		})
		Context("Pod", func() {
			It("create a correct pod", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// check created pod
				pods := &corev1.PodList{}
				assertEntrySize(cl, pods, 1)
				pod := pods.Items[0]

				Ω(pod.ObjectMeta.Labels).Should(HaveKey("app"))
				Ω(pod.ObjectMeta.Labels).Should(HaveKey("created-by"))
				Ω(pod.ObjectMeta.Labels["test-label"]).Should(Equal("foo"))
				Ω(pod.ObjectMeta.Annotations["test-annotation"]).Should(Equal("bar"))
				Ω(pod.ObjectMeta.Annotations["prometheus.io/port"]).Should(Equal(c.DefaultMetricsAddr[:1]))
				Ω(pod.ObjectMeta.Annotations["prometheus.io/scrape"]).Should(Equal("true"))
				Ω(pod.ObjectMeta.Namespace).Should(Equal(el.GetNamespace()))
				Ω(pod.Spec.NodeSelector).Should(HaveLen(1))
				Ω(pod.Spec.NodeSelector["ns-key"]).Should(Equal("ns-value"))

				Ω(pod.Spec.Containers).Should(HaveLen(1))
				container := pod.Spec.Containers[0]
				Ω(*container.Resources.Requests.Cpu()).Should(Equal(defaultPodReqCPU))
				Ω(*container.Resources.Requests.Memory()).Should(Equal(defaultPodReqMem))
				Ω(*container.Resources.Limits.Cpu()).Should(Equal(defaultPodMaxCPU))
				Ω(*container.Resources.Limits.Memory()).Should(Equal(defaultPodMaxMem))

				evars := make(map[string]corev1.EnvVar)
				for _, e := range container.Env {
					evars[e.Name] = e
				}
				Ω(evars[c.EnvWatchNamespace].Value).Should(Equal(ns2))
			})

			It("should update the pod image", func() {
				pod := newPod()
				pod.Spec.Containers[0].Image = "foo"

				cl, _ := testReconcile(el, pod)

				pods := &corev1.PodList{}
				assertEntrySize(cl, pods, 1)
				pod2 := pods.Items[0]
				Ω(pod2.Spec.Containers[0].Image).Should(Equal(testImage))
			})

			It("should use an external service account", func() {
				el.Spec.ServiceAccount = "foo"

				sacc, role, rb := rbacForCR(el)
				cl, _ := testReconcile(el, sacc, role, rb)

				pods := &corev1.PodList{}
				assertEntrySize(cl, pods, 1)
				pod2 := pods.Items[0]

				Ω(pod2.Spec.Containers[0].Image).Should(Equal(testImage))

				assertEntrySize(cl, &corev1.ServiceAccountList{}, 0)
				assertEntrySize(cl, &rbacv1.RoleList{}, 0)
				assertEntrySize(cl, &rbacv1.RoleBindingList{}, 0)
			})
		})
		Context("ServiceAccount", func() {
			It("create a correct service account", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// service account
				saccList := &corev1.ServiceAccountList{}
				assertEntrySize(cl, saccList, 1)
				Ω(saccList.Items[0].ObjectMeta.Name).Should(Equal(loggerName(el)))
			})
		})
		Context("Role", func() {
			It("create a correct role", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// role
				roleList := &rbacv1.RoleList{}
				assertEntrySize(cl, roleList, 1)
				role := roleList.Items[0]
				Ω(role.ObjectMeta.Name).Should(Equal(loggerName(el)))
				Ω(role.Rules).Should(HaveLen(2))
				Ω(role.Rules[0].APIGroups).Should(Equal([]string{""}))
				Ω(role.Rules[0].Resources).Should(Equal([]string{"events", "pods"}))
				Ω(role.Rules[0].Verbs).Should(Equal([]string{"watch", "get", "list"}))

				Ω(role.Rules[1].APIGroups).Should(Equal([]string{"eventlogger.bakito.ch"}))
				Ω(role.Rules[1].Resources).Should(Equal([]string{"eventloggers"}))
				Ω(role.Rules[1].Verbs).Should(Equal([]string{"get", "list", "patch", "update", "watch"}))
			})
		})
		Context("Rolebinding", func() {
			It("create a correct role binding", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// rolebinding
				rbList := &rbacv1.RoleBindingList{}
				assertEntrySize(cl, rbList, 1)
				Ω(rbList.Items[0].ObjectMeta.Name).Should(Equal(loggerName(el)))

				Ω(rbList.Items[0].Subjects).Should(HaveLen(1))
				Ω(rbList.Items[0].Subjects[0].Kind).Should(Equal("ServiceAccount"))
				Ω(rbList.Items[0].Subjects[0].Name).Should(Equal(loggerName(el)))
				Ω(rbList.Items[0].Subjects[0].Namespace).Should(Equal(el.GetNamespace()))
				Ω(rbList.Items[0].RoleRef.Kind).Should(Equal("Role"))
				Ω(rbList.Items[0].RoleRef.Name).Should(Equal(loggerName(el)))
			})
		})
	})
	Context("randString", func() {
		It("generate a random string", func() {
			for i := 0; i < 100; i++ {
				r := randString()
				Ω(r).Should(MatchRegexp("^[a-z]{8}$"))
			}
		})
	})
})

func testReconcile(initialObjects ...client.Object) (client.Client, reconcile.Result) {
	s := scheme.Scheme

	Ω(v1.SchemeBuilder.AddToScheme(s)).ShouldNot(HaveOccurred())

	nn := types.NamespacedName{
		Namespace: uuid.New().String(),
		Name:      uuid.New().String(),
	}
	operatorPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nn.Namespace,
			Name:      nn.Name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: testImage}},
		},
	}
	initialObjects = append(initialObjects, operatorPod)

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(initialObjects...).Build()

	r := &Reconciler{
		Client:           cl,
		Log:              ctrl.Log.WithName("controllers").WithName("Pod"),
		Scheme:           s,
		eventLoggerImage: testImage,
	}

	Ω(r.setupDefaults(cl, nn)).ShouldNot(HaveOccurred())

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(context.Background(), req)
	Ω(err).ShouldNot(HaveOccurred())

	return cl, res
}

func assertEntrySize(cl client.Client, list client.ObjectList, expected int) {
	err := cl.List(context.TODO(), list, client.MatchingLabels{"app": "event-logger-eventlogger"})

	Ω(err).ShouldNot(HaveOccurred())
	r := reflect.ValueOf(list)
	f := reflect.Indirect(r).FieldByName("Items")
	Ω(f.Len()).Should(Equal(expected))
}

func newPod() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":        "event-logger-eventlogger",
				"created-by": "eventlogger",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{},
			},
		},
	}
}
