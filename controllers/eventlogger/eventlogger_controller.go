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

package eventlogger

import (
	"context"
	"fmt"
	"math/rand"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	gracePeriod      int64
	eventLoggerImage = "quay.io/bakito/k8s-event-logger"
	podReqCPU        = resource.MustParse("100m")
	podReqMem        = resource.MustParse("64Mi")
	podMaxCPU        = resource.MustParse("200m")
	podMaxMem        = resource.MustParse("128Mi")
)

// Reconciler reconciles a Pod object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// TODO
// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventlogger.bakito.ch,resources=pods/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
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

	saccChanged, roleChanged, rbChanged, err := r.setupRbac(ctx, cr, reqLogger)
	if err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	// Define a new Pod object
	pod := podForCR(cr)
	// Check if this Pod already exists
	podChanged, err := r.createOrReplacePod(ctx, cr, pod, reqLogger)
	if err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	if saccChanged || roleChanged || rbChanged || podChanged {
		return r.updateCR(ctx, cr, reqLogger, nil)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) createOrReplace(
	ctx context.Context,
	cr *eventloggerv1.EventLogger,
	res runtime.Object,
	reqLogger logr.Logger,
	updateCheck func(curr runtime.Object, next runtime.Object) updateReplace) (bool, error) {
	query := res.DeepCopyObject()
	mo := res.(metav1.Object)
	// Check if this Resource already exists
	err := r.Get(ctx, types.NamespacedName{Name: mo.GetName(), Namespace: mo.GetNamespace()}, query)
	if err != nil && errors.IsNotFound(err) {
		// Set EventLogger cr as the owner and controller
		if err := controllerutil.SetControllerReference(cr, mo, r.Scheme); err != nil {
			return false, err
		}

		reqLogger.Info(fmt.Sprintf("Creating a new %s", query.GetObjectKind().GroupVersionKind().Kind), "namespace", mo.GetNamespace(), "name", mo.GetName())
		err = r.Create(ctx, res.(runtime.Object))
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
			err = r.Update(ctx, res.(runtime.Object))

			if err != nil {
				return false, err
			}
			return true, nil
		} else if check == replace {
			reqLogger.Info(fmt.Sprintf("Replacing %s", query.GetObjectKind().GroupVersionKind().Kind), "namespace", mo.GetNamespace(), "name", mo.GetName())

			err = r.Delete(ctx, query.(runtime.Object))

			if err != nil {
				return false, err
			}
			err = r.Create(ctx, query.(runtime.Object))

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
	err = r.Update(ctx, cr)
	return reconcile.Result{}, err
}

func (r *Reconciler) saveDelete(ctx context.Context, obj runtime.Object) error {
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

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randString() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
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

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch for changes to primary resource EventLogger
	err := ctrl.NewControllerManagedBy(mgr).
		For(&eventloggerv1.EventLogger{}).
		Watches(&source.Kind{Type: &eventloggerv1.EventLogger{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pod and requeue the owner EventLogger
	ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &enqueueDeletedRequestForOwner{
			EnqueueRequestForOwner: handler.EnqueueRequestForOwner{
				IsController: true,
				OwnerType:    &eventloggerv1.EventLogger{},
			},
		}).
		Complete(r)
	
	if err != nil {
		return err
	}
	// Watch for changes to secondary resource Secret and requeue the owner EventLogger
	ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &eventloggerv1.EventLogger{},
		}).
		Complete(r)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

type enqueueDeletedRequestForOwner struct {
	handler.EnqueueRequestForOwner
}

// Create implements Predicate
func (h enqueueDeletedRequestForOwner) Create(event.CreateEvent, workqueue.RateLimitingInterface) {
}

// Delete implements Predicate
func (h enqueueDeletedRequestForOwner) Delete(e event.DeleteEvent, rli workqueue.RateLimitingInterface) {
	h.EnqueueRequestForOwner.Delete(e, rli)
}

// Update implements Predicate
func (h enqueueDeletedRequestForOwner) Update(event.UpdateEvent, workqueue.RateLimitingInterface) {
}

// Generic implements Predicate
func (h enqueueDeletedRequestForOwner) Generic(event.GenericEvent, workqueue.RateLimitingInterface) {
}
