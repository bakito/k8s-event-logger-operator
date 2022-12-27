package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/operator-utils/pkg/filter"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type contextKey string

const (
	defaultContainerName            = "k8s-event-logger-operator"
	configKey            contextKey = "config"
)

var (
	defaultPodReqCPU = resource.MustParse("100m")
	defaultPodReqMem = resource.MustParse("64Mi")
	defaultPodMaxCPU = resource.MustParse("200m")
	defaultPodMaxMem = resource.MustParse("128Mi")
)

// Reconciler reconciles a Pod object
type Reconciler struct {
	client.Reader
	Log    logr.Logger
	Scheme *runtime.Scheme

	cfg              *Cfg
	once             sync.Once
	eventLoggerImage string
}

// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch

// Reconcile EventLogger to setup event logger pods
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)
	return reconcile.Result{}, r.readConfig(ctx, reqLogger, req.NamespacedName)
}

func (r *Reconciler) readConfig(ctx context.Context, reqLogger logr.Logger, nn types.NamespacedName) error {
	// Fetch the EventLogger cr
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, nn, cm)
	if err != nil {
		return err
	}

	reqLogger.Info("Reconciling config")

	noPodTemplate := fmt.Errorf(`configmap %q must contain the container template %q`, nn.String(), cnst.ConfigKeyContainerTemplate)
	if len(cm.Data) == 0 {
		return noPodTemplate
	}
	pt, ok := cm.Data[cnst.ConfigKeyContainerTemplate]
	if !ok {
		return noPodTemplate
	}
	container := corev1.Container{}
	if err := yaml.Unmarshal([]byte(pt), &container); err != nil {
		return err
	}

	if container.Resources.Requests == nil {
		container.Resources.Requests = map[corev1.ResourceName]resource.Quantity{}
	}
	if container.Resources.Limits == nil {
		container.Resources.Limits = map[corev1.ResourceName]resource.Quantity{}
	}

	if _, ok := container.Resources.Requests[corev1.ResourceCPU]; !ok {
		container.Resources.Requests[corev1.ResourceCPU] = defaultPodReqCPU
	}
	if _, ok := container.Resources.Requests[corev1.ResourceMemory]; !ok {
		container.Resources.Requests[corev1.ResourceMemory] = defaultPodReqMem
	}
	if _, ok := container.Resources.Limits[corev1.ResourceCPU]; !ok {
		container.Resources.Limits[corev1.ResourceCPU] = defaultPodMaxCPU
	}
	if _, ok := container.Resources.Limits[corev1.ResourceMemory]; !ok {
		container.Resources.Limits[corev1.ResourceMemory] = defaultPodMaxMem
	}

	if container.Image == "" {
		container.Image = r.eventLoggerImage
	}
	if container.ImagePullPolicy == "" {
		container.ImagePullPolicy = corev1.PullAlways
	}

	r.cfg.ContainerTemplate = container

	return nil
}

func (r *Reconciler) setupEventLoggerImage(nn types.NamespacedName) error {
	if podImage, ok := os.LookupEnv(cnst.EnvEventLoggerImage); ok && podImage != "" {
		r.eventLoggerImage = podImage
		return nil
	}
	p := &corev1.Pod{}
	err := r.Get(context.TODO(), nn, p)
	if err != nil {
		return err
	}

	if len(p.Spec.Containers) == 1 {
		r.eventLoggerImage = p.Spec.Containers[0].Image
		return nil

	}
	for _, c := range p.Spec.Containers {
		if c.Name == defaultContainerName {
			r.eventLoggerImage = c.Image
			return nil
		}
	}
	return fmt.Errorf("could not evaluate the event logger image to use")
}

func (r *Reconciler) Ctx() context.Context {
	r.once.Do(func() {
		r.cfg = &Cfg{}
	})
	return context.WithValue(context.Background(), configKey, r.cfg)
}

func GetCfg(ctx context.Context) *Cfg {
	c, ok := ctx.Value(configKey).(*Cfg)
	if !ok {
		return nil
	}
	clone := *c
	return &clone
}

// SetupWithManager setup with manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	namespace := os.Getenv(cnst.EnvPodNamespace)
	cmName := os.Getenv(cnst.EnvConfigMapName)
	podName := os.Getenv(cnst.EnvPodName)

	if err := r.setupEventLoggerImage(types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	}); err != nil {
		return err
	}

	if err := r.readConfig(r.Ctx(), mgr.GetLogger(), types.NamespacedName{
		Namespace: namespace,
		Name:      cmName,
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).WithEventFilter(filter.NamePredicate{
		Namespace: namespace,
		Names:     []string{cmName},
	},
	).Complete(r)
}

type Cfg struct {
	ContainerTemplate corev1.Container
}
