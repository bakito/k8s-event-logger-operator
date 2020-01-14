package pod

import (
	"context"
	"reflect"
	"testing"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
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
)

func TestPodController(t *testing.T) {
	eventLoggerImage = "quay.io/bakito/k8s-event-logger"
	podReqCPU = resource.MustParse("111m")
	podReqMem = resource.MustParse("222Mi")
	podMaxCPU = resource.MustParse("333m")
	podMaxMem = resource.MustParse("444Mi")

	ns2 := "eventlogger-operators"

	scrape := true
	el := &eventloggerv1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: eventloggerv1.EventLoggerSpec{
			Labels:        map[string]string{"test-label": "foo"},
			Annotations:   map[string]string{"test-annotation": "bar"},
			ScrapeMetrics: &scrape,
			Namespace:     &ns2,
		},
	}

	cl, res := testReconcile(t, el)

	Assert(t, !res.Requeue)

	// check updated status
	updated := &eventloggerv1.EventLogger{}
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
	Assert(t, is.Equal(pod.ObjectMeta.Annotations["prometheus.io/port"], string(c.MetricsPort)))
	Assert(t, is.Equal(pod.ObjectMeta.Annotations["prometheus.io/scrape"], "true"))
	Assert(t, is.Equal(pod.ObjectMeta.Namespace, el.GetNamespace()))

	Assert(t, is.Len(pod.Spec.Containers, 1))
	container := pod.Spec.Containers[0]

	Assert(t, is.Equal(*container.Resources.Requests.Cpu(), podReqCPU))
	Assert(t, is.Equal(*container.Resources.Requests.Memory(), podReqMem))
	Assert(t, is.Equal(*container.Resources.Limits.Cpu(), podMaxCPU))
	Assert(t, is.Equal(*container.Resources.Limits.Memory(), podMaxMem))

	evars := make(map[string]corev1.EnvVar)
	for _, e := range container.Env {
		evars[e.Name] = e
	}
	Assert(t, is.Contains(evars, c.EnvConfigName))
	Assert(t, is.Equal(evars[c.EnvConfigName].Value, el.GetName()))
	Assert(t, is.Equal(evars["WATCH_NAMESPACE"].Value, ns2))

	// role, service account and rolebinding
	saccList := &corev1.ServiceAccountList{}
	assertEntrySize(t, cl, saccList, 1)
	Assert(t, is.Equal(saccList.Items[0].ObjectMeta.Name, loggerName(el)))

	roleList := &rbacv1.RoleList{}
	assertEntrySize(t, cl, roleList, 1)
	Assert(t, is.Equal(roleList.Items[0].ObjectMeta.Name, loggerName(el)))

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
	eventLoggerImage = "quay.io/bakito/k8s-event-logger"
	el := &eventloggerv1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: eventloggerv1.EventLoggerSpec{},
	}
	pod := newPod()
	pod.Spec.Containers[0].Image = "foo"

	cl, _ := testReconcile(t, el, pod)

	pods := &corev1.PodList{}
	assertEntrySize(t, cl, pods, 1)
	pod2 := pods.Items[0]

	Assert(t, is.Equal(pod2.Spec.Containers[0].Image, eventLoggerImage))
}

func TestPodController_extnernal_serviceaccount(t *testing.T) {
	el := &eventloggerv1.EventLogger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventlogger",
			Namespace: testNamespace,
		},
		Spec: eventloggerv1.EventLoggerSpec{
			ServiceAccount: "foo",
		},
	}

	sacc, role, rb := rbacForCR(el)
	cl, _ := testReconcile(t, el, sacc, role, rb)

	pods := &corev1.PodList{}
	assertEntrySize(t, cl, pods, 1)
	pod2 := pods.Items[0]

	Assert(t, is.Equal(pod2.Spec.Containers[0].Image, eventLoggerImage))

	assertEntrySize(t, cl, &corev1.ServiceAccountList{}, 0)
	assertEntrySize(t, cl, &rbacv1.RoleList{}, 0)
	assertEntrySize(t, cl, &rbacv1.RoleBindingList{}, 0)
}

func testReconcile(t *testing.T, intitialObjects ...runtime.Object) (client.Client, reconcile.Result) {

	s := scheme.Scheme
	eventloggerv1.SchemeBuilder.AddToScheme(s)

	cl := fake.NewFakeClient(intitialObjects...)

	r := newReconciler(cl, s)

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

func assertEntrySize(t *testing.T, cl client.Client, list runtime.Object, expectecd int) {
	err := cl.List(context.TODO(), list)
	Assert(t, is.Nil(err))
	r := reflect.ValueOf(list)
	f := reflect.Indirect(r).FieldByName("Items")
	Assert(t, is.Equal(f.Len(), expectecd))
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
