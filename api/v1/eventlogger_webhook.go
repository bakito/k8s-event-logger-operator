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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager setup with manager
func (in *EventLogger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		WithValidator(&validateEl{}).
		For(in).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-eventlogger-bakito-ch-v1-eventlogger,mutating=false,failurePolicy=fail,sideEffects=None,groups=eventlogger.bakito.ch,resources=eventloggers,versions=v1,name=veventlogger.bakito.ch,admissionReviewVersions={v1,v1beta1}

var _ webhook.CustomValidator = &validateEl{}

type validateEl struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *validateEl) ValidateCreate(_ context.Context, o runtime.Object) (warnings admission.Warnings, err error) {
	return v.validate(o)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *validateEl) ValidateUpdate(_ context.Context, o, _ runtime.Object) (warnings admission.Warnings, err error) {
	return v.validate(o)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *validateEl) ValidateDelete(_ context.Context, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *validateEl) validate(obj runtime.Object) (admission.Warnings, error) {
	el, ok := obj.(*EventLogger)
	if !ok {
		return nil, fmt.Errorf("expected a EventLogger but got a %T", obj)
	}

	return nil, el.Spec.Validate()
}
