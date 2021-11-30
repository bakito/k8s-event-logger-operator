/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package setup

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultContainerName = "k8s-event-logger-operator"
)

var (
	gracePeriod      int64
	defaultPodReqCPU = resource.MustParse("100m")
	defaultPodReqMem = resource.MustParse("64Mi")
	defaultPodMaxCPU = resource.MustParse("200m")
	defaultPodMaxMem = resource.MustParse("128Mi")
)

// Reconciler reconciles a Pod object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	eventLoggerImage string
	podReqCPU        resource.Quantity
	podReqMem        resource.Quantity
	podMaxCPU        resource.Quantity
	podMaxMem        resource.Quantity
}

// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=eventloggers,verbs=get;list;watch;create;update;patch;delete

// Reconcile EventLogger to setup event logger pods
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the req.
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	reqLogger.Info("Reconciling event logger")

	if err = cr.Spec.Validate(); err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	saccChanged, roleChanged, rbChanged, err := r.setupRbac(ctx, cr, reqLogger)
	if err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	// Define a new Pod object
	pod := r.podForCR(cr)
	// Check if this Pod already exists
	podChanged, err := r.createOrReplacePod(ctx, cr, pod, reqLogger)
	if err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	if cr.HasChanged() || saccChanged || roleChanged || rbChanged || podChanged {
		return r.updateCR(ctx, cr, reqLogger, nil)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) createOrReplace(
	ctx context.Context,
	cr *eventloggerv1.EventLogger,
	res client.Object,
	reqLogger logr.Logger,
	updateCheck func(curr runtime.Object, next runtime.Object) updateReplace) (bool, error) {
	query := res.DeepCopyObject().(client.Object)
	mo := res.(metav1.Object)
	// Check if this Resource already exists
	err := r.Get(ctx, types.NamespacedName{Name: mo.GetName(), Namespace: mo.GetNamespace()}, query)
	if err != nil && errors.IsNotFound(err) {
		// Set EventLogger cr as the owner and controller
		if err := controllerutil.SetControllerReference(cr, mo, r.Scheme); err != nil {
			return false, err
		}

		reqLogger.Info(fmt.Sprintf("Creating a new %s", query.GetObjectKind().GroupVersionKind().Kind), "namespace", mo.GetNamespace(), "name", mo.GetName())
		err = r.Create(ctx, res)
		if err != nil {
			return false, err
		}
		return true, nil

	} else if err != nil {
		return false, err
	}

	if updateCheck != nil {
		check := updateCheck(query, res)
		if check == update {
			reqLogger.Info(fmt.Sprintf("Updating %s", query.GetObjectKind().GroupVersionKind().Kind), "namespace", mo.GetNamespace(), "name", mo.GetName())
			err = r.Update(ctx, res)

			if err != nil {
				return false, err
			}
			return true, nil
		} else if check == replace {
			reqLogger.Info(fmt.Sprintf("Replacing %s", query.GetObjectKind().GroupVersionKind().Kind), "namespace", mo.GetNamespace(), "name", mo.GetName())

			err = r.Delete(ctx, query)

			if err != nil {
				return false, err
			}
			err = r.Create(ctx, query)

			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	// Resource already exists
	return false, nil
}

func (r *Reconciler) updateCR(ctx context.Context, cr *eventloggerv1.EventLogger, logger logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		logger.Error(err, "")
	}
	cr.Apply(err)
	cr.Status.Hash = cr.Spec.Hash()
	cr.Status.OperatorVersion = version.Version
	err = r.Update(ctx, cr)
	return reconcile.Result{}, err
}

func (r *Reconciler) saveDelete(ctx context.Context, obj client.Object) error {
	err := r.Delete(ctx, obj)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

type updateReplace string

const (
	update  updateReplace = "update"
	replace updateReplace = "replace"
	no      updateReplace = "no"
)

func randString() string {
	var result []byte
	for {
		if len(result) >= 8 {
			return string(result)
		}
		num, err := rand.Int(rand.Reader, big.NewInt(int64(127)))
		if err != nil {
			return ""
		}
		n := num.Int64()
		if n >= 97 && n <= 122 {
			result = append(result, byte(n))
		}
	}
}

func loggerName(cr *eventloggerv1.EventLogger) string {
	return "event-logger-" + cr.Name
}

func podChanged(old, new *corev1.Pod) bool {
	if old.Spec.ServiceAccountName != new.Spec.ServiceAccountName {
		return true
	}
	if len(old.Spec.Containers) > 0 && len(new.Spec.Containers) > 0 && old.Spec.Containers[0].Image != new.Spec.Containers[0].Image {
		return true
	}

	return podEnv(old, "WATCH_NAMESPACE") != podEnv(new, "WATCH_NAMESPACE")
}

func podEnv(pod *corev1.Pod, name string) string {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == name {
			return env.Value
		}
	}
	return "N/A"
}

// SetupWithManager setup with manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.setupDefaults(mgr.GetAPIReader(), types.NamespacedName{
		Namespace: os.Getenv(cnst.EnvPodNamespace),
		Name:      os.Getenv(cnst.EnvPodName),
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&eventloggerv1.EventLogger{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

func (r *Reconciler) setupDefaults(client client.Reader, nn types.NamespacedName) error {
	if cpu, ok := os.LookupEnv(cnst.EnvLoggerPodReqCPU); ok {
		r.podReqCPU = resource.MustParse(cpu)
	} else {
		r.podReqCPU = defaultPodReqCPU
	}
	if mem, ok := os.LookupEnv(cnst.EnvLoggerPodReqMem); ok {
		r.podReqMem = resource.MustParse(mem)
	} else {
		r.podReqMem = defaultPodReqMem
	}
	if cpu, ok := os.LookupEnv(cnst.EnvLoggerPodMaxCPU); ok {
		r.podMaxCPU = resource.MustParse(cpu)
	} else {
		r.podMaxCPU = defaultPodMaxCPU
	}
	if mem, ok := os.LookupEnv(cnst.EnvLoggerPodMaxMem); ok {
		r.podMaxMem = resource.MustParse(mem)
	} else {
		r.podMaxMem = defaultPodMaxMem
	}

	return r.setupEventLoggerImage(client, nn)
}

func (r *Reconciler) setupEventLoggerImage(client client.Reader, nn types.NamespacedName) error {
	if podImage, ok := os.LookupEnv(cnst.EnvEventLoggerImage); ok && podImage != "" {
		r.eventLoggerImage = podImage
		return nil
	}
	p := &corev1.Pod{}
	err := client.Get(context.TODO(), nn, p)
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
