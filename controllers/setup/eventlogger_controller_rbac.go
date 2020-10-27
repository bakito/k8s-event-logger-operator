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
	"reflect"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (r *Reconciler) setupRbac(ctx context.Context, cr *eventloggerv1.EventLogger, reqLogger logr.Logger) (bool, bool, bool, error) {
	var saccChanged, roleChanged, rbChanged bool
	var err error
	sacc, role, rb := rbacForCR(cr)

	if cr.Spec.ServiceAccount == "" {
		saccChanged, err = r.createOrReplace(ctx, cr, sacc, reqLogger, nil)
		if err != nil {
			return saccChanged, roleChanged, rbChanged, err
		}
		roleChanged, err = r.createOrReplace(ctx, cr, role, reqLogger, func(curr runtime.Object, next runtime.Object) updateReplace {
			o1 := curr.(*rbacv1.Role)
			o2 := next.(*rbacv1.Role)
			if reflect.DeepEqual(o1.Rules, o2.Rules) {
				return no
			}
			return update
		})
		if err != nil {
			return saccChanged, roleChanged, rbChanged, err
		}
		rbChanged, err = r.createOrReplace(ctx, cr, rb, reqLogger, nil)
		if err != nil {
			return saccChanged, roleChanged, rbChanged, err
		}
	} else {
		// Only delete sa if the name is different than the configured
		if cr.Spec.ServiceAccount != sacc.GetName() {
			err = r.saveDelete(ctx, sacc)
			if err != nil {
				return saccChanged, roleChanged, rbChanged, err
			}
		}
		err = r.saveDelete(ctx, role)
		if err != nil {
			return saccChanged, roleChanged, rbChanged, err
		}
		err = r.saveDelete(ctx, rb)
		if err != nil {
			return saccChanged, roleChanged, rbChanged, err
		}
	}
	return saccChanged, roleChanged, rbChanged, nil
}

func rbacForCR(cr *eventloggerv1.EventLogger) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding) {
	sacc := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": loggerName(cr),
			},
		},
	}

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind: "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": loggerName(cr),
			},
		},
		Rules: []rbacv1.PolicyRule{
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
		},
	}
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind: "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggerName(cr),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": loggerName(cr),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      loggerName(cr),
				Namespace: cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     loggerName(cr),
		},
	}

	return sacc, role, rb
}
