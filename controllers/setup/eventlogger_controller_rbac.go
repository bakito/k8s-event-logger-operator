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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) setupRbac(ctx context.Context, cr *eventloggerv1.EventLogger) (bool, bool, bool, error) {
	var err error
	sacc, role, rb := rbacForCR(cr)

	if cr.Spec.ServiceAccount == "" {
		saccRes, err := controllerutil.CreateOrUpdate(ctx, r.Client, sacc, func() error {
			if sacc.Labels == nil {
				sacc.Labels = map[string]string{}
			}
			sacc.Labels["app"] = loggerName(cr)
			return nil
		})
		if err != nil {
			return false, false, false, err
		}

		roleRes, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
			if role.Labels == nil {
				role.Labels = map[string]string{}
			}
			role.Labels["app"] = loggerName(cr)
			role.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"events", "pods"},
					Verbs:     []string{"watch", "get", "list"},
				},
				{
					APIGroups: []string{"eventlogger.bakito.ch"},
					Resources: []string{"eventloggers"},
					Verbs:     []string{"get", "list", "patch", "update", "watch"},
				},
			}
			return nil
		})
		if err != nil {
			return false, false, false, err
		}

		rbRes, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
			if rb.Labels == nil {
				rb.Labels = map[string]string{}
			}
			rb.Labels["app"] = loggerName(cr)

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      loggerName(cr),
					Namespace: cr.Namespace,
				},
			}
			rb.RoleRef = rbacv1.RoleRef{
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     loggerName(cr),
			}
			return nil
		})
		if err != nil {
			return false, false, false, err
		}
		return saccRes != controllerutil.OperationResultNone, roleRes != controllerutil.OperationResultNone, rbRes != controllerutil.OperationResultNone, nil
	}

	// Only delete sa if the name is different from the configured
	if cr.Spec.ServiceAccount != sacc.GetName() {
		err = r.saveDelete(ctx, sacc)
		if err != nil {
			return false, false, false, err
		}
	}
	err = r.saveDelete(ctx, role)
	if err != nil {
		return false, false, false, err
	}
	err = r.saveDelete(ctx, rb)
	if err != nil {
		return false, false, false, err
	}
	return false, false, false, nil
}

func rbacForCR(cr *eventloggerv1.EventLogger) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding) {
	sacc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
		},
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
		},
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
		},
	}

	return sacc, role, rb
}
