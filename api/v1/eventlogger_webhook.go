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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var eventloggerlog = logf.Log.WithName("eventlogger-resource")

// SetupWebhookWithManager setup with manager
func (r *EventLogger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-eventlogger-bakito-ch-v1-eventlogger,mutating=true,failurePolicy=fail,groups=eventlogger.bakito.ch,resources=eventloggers,verbs=create;update,versions=v1,name=meventlogger.kb.io

var _ webhook.Defaulter = &EventLogger{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EventLogger) Default() {
	eventloggerlog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-eventlogger-bakito-ch-v1-eventlogger,mutating=false,failurePolicy=fail,groups=eventlogger.bakito.ch,resources=eventloggers,versions=v1,name=veventlogger.kb.io

var _ webhook.Validator = &EventLogger{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EventLogger) ValidateCreate() error {
	eventloggerlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EventLogger) ValidateUpdate(old runtime.Object) error {
	eventloggerlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EventLogger) ValidateDelete() error {
	eventloggerlog.Info("validate delete", "name", r.Name)

	return nil
}
