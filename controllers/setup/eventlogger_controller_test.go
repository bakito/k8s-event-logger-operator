package setup

import (
	"context"
	"reflect"
	"time"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/controllers/config"
	c "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/version"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
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
		ns2 = "eventlogger-operators"

		el = &v1.EventLogger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "eventlogger",
				Namespace: testNamespace,
			},
			Spec: v1.EventLoggerSpec{
				Labels:        map[string]string{"test-label": "foo"},
				Annotations:   map[string]string{"test-annotation": "bar"},
				ScrapeMetrics: ptr.To(true),
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
				assertEntrySize(cl, el, pods, 1)
				pod := pods.Items[0]

				Ω(pod.ObjectMeta.Labels).Should(HaveKey(labelComponent))
				Ω(pod.ObjectMeta.Labels).Should(HaveKey(labelManagedBy))
				Ω(pod.ObjectMeta.Labels["test-label"]).Should(Equal("foo"))
				Ω(pod.ObjectMeta.Annotations["test-annotation"]).Should(Equal("bar"))
				Ω(pod.ObjectMeta.Annotations["prometheus.io/port"]).Should(Equal(c.DefaultMetricsAddr[:1]))
				Ω(pod.ObjectMeta.Annotations["prometheus.io/scrape"]).Should(Equal("true"))
				Ω(pod.ObjectMeta.Namespace).Should(Equal(el.GetNamespace()))
				Ω(pod.ObjectMeta.OwnerReferences).Should(HaveLen(1))

				Ω(pod.Spec.NodeSelector).Should(HaveLen(1))
				Ω(pod.Spec.NodeSelector["ns-key"]).Should(Equal("ns-value"))

				Ω(pod.Spec.Containers).Should(HaveLen(1))
				container := pod.Spec.Containers[0]
				Ω(*container.Resources.Requests.Cpu()).Should(Equal(resource.MustParse("111m")))
				Ω(*container.Resources.Requests.Memory()).Should(Equal(resource.MustParse("222Mi")))
				Ω(*container.Resources.Limits.Cpu()).Should(Equal(resource.MustParse("333m")))
				Ω(*container.Resources.Limits.Memory()).Should(Equal(resource.MustParse("444Mi")))

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
				assertEntrySize(cl, el, pods, 1)
				pod2 := pods.Items[0]
				Ω(pod2.Spec.Containers[0].Image).Should(Equal(testImage))
			})

			It("should update the imagePullSecrets", func() {
				el.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "secret1"}, {Name: "secret2"}}

				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				pods := &corev1.PodList{}
				assertEntrySize(cl, el, pods, 1)
				pod2 := pods.Items[0]

				Ω(len(pod2.Spec.ImagePullSecrets)).Should(Equal(2))
				Ω(pod2.Spec.ImagePullSecrets[0].Name).Should(Equal("secret1"))
				Ω(pod2.Spec.ImagePullSecrets[1].Name).Should(Equal("secret2"))
			})

			It("should use an external service account", func() {
				el.Spec.ServiceAccount = "foo"

				sacc, role, rb := rbacForCR(el)
				cl, _ := testReconcile(el, sacc, role, rb)

				pods := &corev1.PodList{}
				assertEntrySize(cl, el, pods, 1)
				pod2 := pods.Items[0]

				Ω(pod2.Spec.Containers[0].Image).Should(Equal(testImage))

				assertEntrySize(cl, el, &corev1.ServiceAccountList{}, 0)
				assertEntrySize(cl, el, &rbacv1.RoleList{}, 0)
				assertEntrySize(cl, el, &rbacv1.RoleBindingList{}, 0)
			})
		})
		Context("ServiceAccount", func() {
			It("create a correct service account", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// service account
				saccList := &corev1.ServiceAccountList{}
				assertEntrySize(cl, el, saccList, 1)
				sacc := saccList.Items[0]
				Ω(sacc.ObjectMeta.Name).Should(Equal(loggerName(el)))
				Ω(sacc.ObjectMeta.Labels).Should(HaveKey(labelComponent))
				Ω(sacc.ObjectMeta.Labels).Should(HaveKey(labelManagedBy))
				Ω(sacc.ObjectMeta.OwnerReferences).Should(HaveLen(1))
			})
		})
		Context("Role", func() {
			It("create a correct role", func() {
				cl, res := testReconcile(el)
				Ω(res.Requeue).Should(BeFalse())

				// role
				roleList := &rbacv1.RoleList{}
				assertEntrySize(cl, el, roleList, 1)
				role := roleList.Items[0]
				Ω(role.ObjectMeta.Name).Should(Equal(loggerName(el)))
				Ω(role.ObjectMeta.Labels).Should(HaveKey(labelComponent))
				Ω(role.ObjectMeta.Labels).Should(HaveKey(labelManagedBy))
				Ω(role.ObjectMeta.OwnerReferences).Should(HaveLen(1))

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
				assertEntrySize(cl, el, rbList, 1)
				rb := rbList.Items[0]
				Ω(rb.ObjectMeta.Name).Should(Equal(loggerName(el)))
				Ω(rb.ObjectMeta.Labels).Should(HaveKey(labelComponent))
				Ω(rb.ObjectMeta.Labels).Should(HaveKey(labelManagedBy))
				Ω(rb.ObjectMeta.OwnerReferences).Should(HaveLen(1))

				Ω(rb.Subjects).Should(HaveLen(1))
				Ω(rb.Subjects[0].Kind).Should(Equal("ServiceAccount"))
				Ω(rb.Subjects[0].Name).Should(Equal(loggerName(el)))
				Ω(rb.Subjects[0].Namespace).Should(Equal(el.GetNamespace()))
				Ω(rb.RoleRef.Kind).Should(Equal("Role"))
				Ω(rb.RoleRef.Name).Should(Equal(loggerName(el)))
			})
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
	cfg := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nn.Namespace,
			Name:      nn.Name,
		},
		Data: map[string]string{c.ConfigKeyContainerTemplate: `
image: quay.io/bakito/k8s-event-logger
resources:
  limits:
    cpu: 333m
    memory: 444Mi
  requests:
    cpu: 111m
    memory: 222Mi
`},
	}

	initialObjects = append(initialObjects, operatorPod, cfg)

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(initialObjects...).Build()

	cr := config.Reconciler{
		Reader: cl,
		Log:    ctrl.Log.WithName("controllers").WithName("Pod"),
		Scheme: s,
	}

	_, err := cr.Reconcile(cr.Ctx(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
		},
	})
	Ω(err).ShouldNot(HaveOccurred())

	r := &Reconciler{
		Client: cl,
		Log:    ctrl.Log.WithName("controllers").WithName("Pod"),
		Scheme: s,
		Config: cr.Ctx(),
	}

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

func assertEntrySize(cl client.Client, el *v1.EventLogger, list client.ObjectList, expected int) {
	option := client.MatchingLabels{}
	applyDefaultLabels(el, option)
	err := cl.List(context.TODO(), list, option)

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
