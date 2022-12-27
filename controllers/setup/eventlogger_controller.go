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

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var gracePeriod int64

// Reconciler reconciles a Pod object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	Config context.Context
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

	saccChanged, roleChanged, rbChanged, err := r.setupRbac(ctx, cr)
	if err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

	// Define a new Pod object
	pod := r.podForCR(cr)

	// set owner reference for pod
	if err := controllerutil.SetOwnerReference(cr, pod, r.Scheme); err != nil {
		return r.updateCR(ctx, cr, reqLogger, err)
	}

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
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventloggerv1.EventLogger{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
