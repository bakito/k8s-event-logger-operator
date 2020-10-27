package setup

import (
	"context"
	"github.com/google/uuid"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	c "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/version"
	. "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "eventlogger-operator"
	testImage     = "quay.io/bakito/k8s-event-logger"
)

func TestPodController(t *testing.T) {
	defaultPodReqCPU = resource.MustParse("111m")
	defaultPodReqMem = resource.MustParse("222Mi")
	defaultPodMaxCPU = resource.MustParse("333m")
	defaultPodMaxMem = resource.MustParse("444Mi")

	ns2 := "eventlogger-operators"

	scrape := true
	el := &v1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: v1.EventLoggerSpec{
			Labels:        map[string]string{"test-label": "foo"},
			Annotations:   map[string]string{"test-annotation": "bar"},
			ScrapeMetrics: &scrape,
			Namespace:     &ns2,
		},
	}

	cl, res := testReconcile(t, el)

	Assert(t, !res.Requeue)

	// check updated status
	updated := &v1.EventLogger{}
	err := cl.Get(context.TODO(), types.NamespacedName{
		Name:      "eventlogger",
		Namespace: testNamespace,
	}, updated)
	Assert(t, is.Nil(err))
	Assert(t, updated.Status.LastProcessed.String() != "")
	Assert(t, is.Equal(updated.Status.OperatorVersion, version.Version))

	// check created pod
	pods := &corev1.PodList{}
	assertEntrySize(t, cl, pods, 1)
	pod := pods.Items[0]

	Assert(t, is.Contains(pod.ObjectMeta.Labels, "app"))
	Assert(t, is.Contains(pod.ObjectMeta.Labels, "created-by"))
	Assert(t, is.Equal(pod.ObjectMeta.Labels["test-label"], "foo"))
	Assert(t, is.Equal(pod.ObjectMeta.Annotations["test-annotation"], "bar"))
	Assert(t, is.Equal(pod.ObjectMeta.Annotations["prometheus.io/port"], string(c.DefaultMetricsAddr[:1])))
	Assert(t, is.Equal(pod.ObjectMeta.Annotations["prometheus.io/scrape"], "true"))
	Assert(t, is.Equal(pod.ObjectMeta.Namespace, el.GetNamespace()))

	Assert(t, is.Len(pod.Spec.Containers, 1))
	container := pod.Spec.Containers[0]

	Assert(t, is.Equal(*container.Resources.Requests.Cpu(), defaultPodReqCPU))
	Assert(t, is.Equal(*container.Resources.Requests.Memory(), defaultPodReqMem))
	Assert(t, is.Equal(*container.Resources.Limits.Cpu(), defaultPodMaxCPU))
	Assert(t, is.Equal(*container.Resources.Limits.Memory(), defaultPodMaxMem))

	evars := make(map[string]corev1.EnvVar)
	for _, e := range container.Env {
		evars[e.Name] = e
	}
	Assert(t, is.Equal(evars[c.EnvWatchNamespace].Value, ns2))

	// service account
	saccList := &corev1.ServiceAccountList{}
	assertEntrySize(t, cl, saccList, 1)
	Assert(t, is.Equal(saccList.Items[0].ObjectMeta.Name, loggerName(el)))

	// role
	roleList := &rbacv1.RoleList{}
	assertEntrySize(t, cl, roleList, 1)
	role := roleList.Items[0]
	Assert(t, is.Equal(role.ObjectMeta.Name, loggerName(el)))
	Assert(t, is.Len(role.Rules, 2))
	Assert(t, is.DeepEqual(role.Rules[0].APIGroups, []string{""}))
	Assert(t, is.DeepEqual(role.Rules[0].Resources, []string{"events", "pods"}))
	Assert(t, is.DeepEqual(role.Rules[0].Verbs, []string{"watch", "get", "list"}))

	Assert(t, is.DeepEqual(role.Rules[1].APIGroups, []string{"eventlogger.bakito.ch"}))
	Assert(t, is.DeepEqual(role.Rules[1].Resources, []string{"eventloggers"}))
	Assert(t, is.DeepEqual(role.Rules[1].Verbs, []string{"get", "list", "patch", "update", "watch"}))

	// rolebinding
	rbList := &rbacv1.RoleBindingList{}
	assertEntrySize(t, cl, rbList, 1)
	Assert(t, is.Equal(rbList.Items[0].ObjectMeta.Name, loggerName(el)))

	Assert(t, is.Len(rbList.Items[0].Subjects, 1))
	Assert(t, is.Equal(rbList.Items[0].Subjects[0].Kind, "ServiceAccount"))
	Assert(t, is.Equal(rbList.Items[0].Subjects[0].Name, loggerName(el)))
	Assert(t, is.Equal(rbList.Items[0].Subjects[0].Namespace, el.GetNamespace()))

	Assert(t, is.Equal(rbList.Items[0].RoleRef.Kind, "Role"))
	Assert(t, is.Equal(rbList.Items[0].RoleRef.Name, loggerName(el)))
}

func TestPodController_changed_image(t *testing.T) {
	el := &v1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: v1.EventLoggerSpec{},
	}
	pod := newPod()
	pod.Spec.Containers[0].Image = "foo"

	cl, _ := testReconcile(t, el, pod)

	pods := &corev1.PodList{}
	assertEntrySize(t, cl, pods, 1)
	pod2 := pods.Items[0]

	Assert(t, is.Equal(pod2.Spec.Containers[0].Image, testImage))
}

func TestPodController_extnernal_serviceaccount(t *testing.T) {
	el := &v1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: v1.EventLoggerSpec{
			ServiceAccount: "foo",
		},
	}

	sacc, role, rb := rbacForCR(el)
	cl, _ := testReconcile(t, el, sacc, role, rb)

	pods := &corev1.PodList{}
	assertEntrySize(t, cl, pods, 1)
	pod2 := pods.Items[0]

	Assert(t, is.Equal(pod2.Spec.Containers[0].Image, testImage))

	assertEntrySize(t, cl, &corev1.ServiceAccountList{}, 0)
	assertEntrySize(t, cl, &rbacv1.RoleList{}, 0)
	assertEntrySize(t, cl, &rbacv1.RoleBindingList{}, 0)
}

func testReconcile(t *testing.T, intitialObjects ...runtime.Object) (client.Client, reconcile.Result) {

	s := scheme.Scheme
	Assert(t, is.Nil(v1.SchemeBuilder.AddToScheme(s)))

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
	intitialObjects = append(intitialObjects, operatorPod)

	cl := fake.NewFakeClientWithScheme(s, intitialObjects...)

	r := &Reconciler{
		Client:           cl,
		Log:              ctrl.Log.WithName("controllers").WithName("Pod"),
		Scheme:           s,
		eventLoggerImage: testImage,
	}

	Assert(t, is.Nil(r.setupDefaults(cl, nn)))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	Assert(t, is.Nil(err))

	return cl, res
}

func assertEntrySize(t *testing.T, cl client.Client, list runtime.Object, expected int) {
	err := cl.List(context.TODO(), list, client.MatchingLabels{"app": "event-logger-eventlogger"})
	Assert(t, is.Nil(err))
	r := reflect.ValueOf(list)
	f := reflect.Indirect(r).FieldByName("Items")
	Assert(t, is.Equal(f.Len(), expected))
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
