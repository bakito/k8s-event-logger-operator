package config

import (
	"os"
	"time"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Config", func() {
	var (
		s  *runtime.Scheme
		cr *Reconciler
	)

	BeforeEach(func() {
		s = scheme.Scheme
		Ω(v1.SchemeBuilder.AddToScheme(s)).ShouldNot(HaveOccurred())

		cr = &Reconciler{
			Log:    ctrl.Log.WithName("controllers").WithName("Pod"),
			Scheme: s,
		}
	})

	Context("Reconcile", func() {
		var configMap *corev1.ConfigMap
		BeforeEach(func() {
			configMap = &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: uuid.NewString(),
			}}
		})

		It("should fail if the data is empty", func() {
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(configMap).Build()
			res, err := cr.Reconcile(cr.Ctx(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				},
			})
			Ω(err).Should(HaveOccurred())
			Ω(res.RequeueAfter).Should(Equal(time.Duration(0)))
		})
		It("should fail if the container template does not exist", func() {
			configMap.Data = map[string]string{"foo": "bar"}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(configMap).Build()
			res, err := cr.Reconcile(cr.Ctx(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				},
			})
			Ω(err).Should(HaveOccurred())
			Ω(res.RequeueAfter).Should(Equal(time.Duration(0)))
		})
		It("should read the default config", func() {
			configMap.Data = map[string]string{cnst.ConfigKeyContainerTemplate: ""}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(configMap).Build()
			res, err := cr.Reconcile(cr.Ctx(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				},
			})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).Should(Equal(time.Duration(0)))
			cfg := GetCfg(cr.Ctx())

			Ω(cfg.ContainerTemplate.Resources.Requests.Cpu().String()).Should(Equal(defaultPodReqCPU.String()))
			Ω(cfg.ContainerTemplate.Resources.Requests.Memory().String()).Should(Equal(defaultPodReqMem.String()))
			Ω(cfg.ContainerTemplate.Resources.Limits.Cpu().String()).Should(Equal(defaultPodMaxCPU.String()))
			Ω(cfg.ContainerTemplate.Resources.Limits.Memory().String()).Should(Equal(defaultPodMaxMem.String()))
		})
		It("should fail if the container template can not be parsed", func() {
			configMap.Data = map[string]string{cnst.ConfigKeyContainerTemplate: `
image: quay.io/bakito/k8s-event-logger
resources:
  requests:
    cpu: 111m
    memory: 222Mi
  limits:
    cpu: 333m
    memory: 444Mi
`}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(configMap).Build()
			res, err := cr.Reconcile(cr.Ctx(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				},
			})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).Should(Equal(time.Duration(0)))

			cfg := GetCfg(cr.Ctx())
			Ω(cfg.ContainerTemplate.Resources.Requests.Cpu().String()).Should(Equal("111m"))
			Ω(cfg.ContainerTemplate.Resources.Requests.Memory().String()).Should(Equal("222Mi"))
			Ω(cfg.ContainerTemplate.Resources.Limits.Cpu().String()).Should(Equal("333m"))
			Ω(cfg.ContainerTemplate.Resources.Limits.Memory().String()).Should(Equal("444Mi"))
		})
	})

	Context("setupEventLoggerImage", func() {
		var (
			pod *corev1.Pod
			nn  types.NamespacedName
		)
		BeforeEach(func() {
			pod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: uuid.NewString(),
			}}
			nn = types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}
		})
		AfterEach(func() {
			_ = os.Setenv(cnst.EnvEventLoggerImage, "")
		})

		It("should evaluate the image from the env variable", func() {
			_ = os.Setenv(cnst.EnvEventLoggerImage, "my:image")
			err := cr.setupEventLoggerImage(nn)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cr.eventLoggerImage).Should(Equal("my:image"))
		})
		It("should fail if no pod is found", func() {
			cr.Reader = fake.NewClientBuilder().WithScheme(s).Build()
			err := cr.setupEventLoggerImage(nn)
			Ω(err).Should(HaveOccurred())
			Ω(cr.eventLoggerImage).Should(BeEmpty())
		})
		It("should evaluate the image from the first operator pod container", func() {
			pod.Spec = corev1.PodSpec{Containers: []corev1.Container{
				{Image: "my-container:image"},
			}}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(pod).Build()
			err := cr.setupEventLoggerImage(nn)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cr.eventLoggerImage).Should(Equal("my-container:image"))
		})

		It("should evaluate the image from the operator pod container", func() {
			pod.Spec = corev1.PodSpec{Containers: []corev1.Container{
				{Image: "my-container:image1", Name: "something-else"},
				{Image: "my-container:image2", Name: defaultContainerName},
			}}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(pod).Build()
			err := cr.setupEventLoggerImage(nn)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cr.eventLoggerImage).Should(Equal("my-container:image2"))
		})
		It("should fail if the container is not found", func() {
			pod.Spec = corev1.PodSpec{Containers: []corev1.Container{
				{Image: "my-container:image1", Name: "something-else"},
				{Image: "my-container:image2", Name: "not-my-container"},
			}}
			cr.Reader = fake.NewClientBuilder().WithScheme(s).WithObjects(pod).Build()
			err := cr.setupEventLoggerImage(nn)
			Ω(err).Should(HaveOccurred())
			Ω(cr.eventLoggerImage).Should(BeEmpty())
		})
	})
})
