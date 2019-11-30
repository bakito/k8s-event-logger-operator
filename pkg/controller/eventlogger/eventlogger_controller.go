package eventlogger

import (
	"context"
	"reflect"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	"sigs.k8s.io/yaml"
)

var (
	log                   = logf.Log.WithName("controller_eventlogger")
	replicas        int32 = 1
	defaultFileMode int32 = 420
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

	// Fetch the EventLogger cr
	cr := &eventloggerv1.EventLogger{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cr)
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
	dep := deploymentForCR(cr, nil)

	// Set EventLogger cr as the owner and controller
	if err := controllerutil.SetControllerReference(cr, dep, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)

		conf, err := yaml.Marshal(cr.Spec)
		if err != nil {
			return reconcile.Result{}, err
		}
		sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name + "-event-logger",
				Namespace: cr.Namespace,
				Labels: map[string]string{
					"app": cr.Name,
				},
			},
			Type: "github.com/bakito/k8s-event-logger-operator",
			Data: map[string][]byte{
				"event-listener": conf,
			},
		}
		// Set EventLogger cr as the owner and controller
		if err := controllerutil.SetControllerReference(cr, sec, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		err = r.client.Create(context.TODO(), sec)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return reconcile.Result{}, err
		}

		sacc, role, rb, err := rbac(cr, r.scheme)
		if err != nil {
			return reconcile.Result{}, err
		}

		err = r.client.Create(context.TODO(), sacc)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Create(context.TODO(), role)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Create(context.TODO(), rb)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Deployment created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Deployment already exists - check if changed
	if checkChanged(cr, found) {
		reqLogger.Info("Updating Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// check if the existing deployment was changed
func checkChanged(cr *eventloggerv1.EventLogger, dep *appsv1.Deployment) bool {
	clone := dep.DeepCopy()
	// re-apply initial values to original dep
	deploymentForCR(cr, dep)

	return !reflect.DeepEqual(dep, clone)
}

// deploymentForCR returns a busybox dep with the same name/namespace as the cr
func deploymentForCR(cr *eventloggerv1.EventLogger, dep *appsv1.Deployment) *appsv1.Deployment {
	if dep == nil {
		dep = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									corev1.EnvVar{},
								},
								VolumeMounts: []corev1.VolumeMount{
									corev1.VolumeMount{},
								},
							},
						},
						Volumes: []corev1.Volume{
							corev1.Volume{
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{},
								},
							},
						},
					},
				},
			},
		}
	}
	labels := map[string]string{
		"app": cr.Name,
	}

	dep.ObjectMeta.Name = cr.Name + "-event-logger"
	dep.ObjectMeta.Namespace = cr.Namespace
	dep.ObjectMeta.Labels = labels
	dep.Spec.Replicas = &replicas
	dep.Spec.Selector.MatchLabels = labels
	dep.Spec.Template.ObjectMeta.Labels = labels
	dep.Spec.Template.Spec.Containers[0].Name = "event-logger"
	dep.Spec.Template.Spec.Containers[0].Image = "quay.io/bakito/k8s-event-logger"
	dep.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "SLEEP",
			Value: "1000",
		},
	}

	dep.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "config",
			MountPath: "/opt/go/config",
		},
	}
	dep.Spec.Template.Spec.Volumes = []corev1.Volume{
		corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &defaultFileMode,
					SecretName:  cr.Name + "-event-logger",
				},
			},
		},
	}

	return dep
}

func rbac(cr *eventloggerv1.EventLogger, scheme *runtime.Scheme) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, error) {
	sacc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
	}

	// Set EventLogger cr as the owner and controller
	if err := controllerutil.SetControllerReference(cr, sacc, scheme); err != nil {
		return nil, nil, nil, err
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"watch", "get", "list"},
			},
		},
	}

	// Set EventLogger cr as the owner and controller
	if err := controllerutil.SetControllerReference(cr, role, scheme); err != nil {
		return nil, nil, nil, err
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      cr.Name,
				Namespace: cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     cr.Name,
		},
	}

	// Set EventLogger cr as the owner and controller
	err := controllerutil.SetControllerReference(cr, rb, scheme)
	return sacc, role, rb, err
}
