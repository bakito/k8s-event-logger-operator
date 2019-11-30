package eventlogger

import (
	"context"

	eventloggerv1 "github.com/bakito/event-logger-operator/pkg/apis/eventlogger/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log            = logf.Log.WithName("controller_eventlogger")
	replicas int32 = 1
)

// Add creates a new EventLogger Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEventLogger{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("eventlogger-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EventLogger
	err = c.Watch(&source.Kind{Type: &eventloggerv1.EventLogger{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner EventLogger
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &eventloggerv1.EventLogger{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileEventLogger implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileEventLogger{}

// ReconcileEventLogger reconciles a EventLogger object
type ReconcileEventLogger struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a EventLogger object and makes changes based on the state read
// and what is in the EventLogger.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEventLogger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling EventLogger")

	// Fetch the EventLogger instance
	instance := &eventloggerv1.EventLogger{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Deployment object
	dep := newDeploymentForCR(instance)

	// Set EventLogger instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Deployment created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Deployment already exists - check if changed
	if checkChanged(dep) {
		reqLogger.Info("Updating Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Update(context.TODO(), dep)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		reqLogger.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	}

	return reconcile.Result{}, nil
}

// check if the existing deployment was changed
func checkChanged(dep *appsv1.Deployment) bool {
	if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 1 {
		dep.Spec.Replicas = &replicas
		return true
	}
	return false
}

// newDeploymentForCR returns a busybox dep with the same name/namespace as the cr
func newDeploymentForCR(cr *eventloggerv1.EventLogger) *appsv1.Deployment {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-event-logger",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "busybox",
							Image:   "busybox",
							Command: []string{"sleep", "3600"},
						},
					},
				},
			},
		},
	}
}
