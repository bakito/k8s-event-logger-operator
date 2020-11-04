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
	"github.com/bakito/k8s-event-logger-operator/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventLoggerSpec defines the desired state of EventLogger
type EventLoggerSpec struct {

	// Kinds the kinds to log the events for
	// +kubebuilder:validation:MinItems=1
	Kinds []Kind `json:"kinds,omitempty"`

	// EventTypes the event types to log. If empty all events are logged.
	// +kubebuilder:validation:MinItems=0
	EventTypes []string `json:"eventTypes,omitempty"`

	// Labels additional labels for the logger pod
	Labels map[string]string `json:"labels,omitempty"`

	// Labels additional annotations for the logger pod
	Annotations map[string]string `json:"annotations,omitempty"`

	// ScrapeMetrics if true, prometheus scrape annotations are added to the pod
	ScrapeMetrics *bool `json:"scrapeMetrics,omitempty"`

	// namespace the namespace to watch on, may be an empty string
	// +nullable
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// ServiceAccount the service account to use for the logger pod
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// LogFields fields ot the event to be logged.
	LogFields []LogField `json:"logFields,omitempty"`
}

// Kind defines a kind to log events for
type Kind struct {
	// +kubebuilder:validation:MinLength=3
	Name string `json:"name"`

	// EventTypes the event types to log. If empty events are logged as defined in spec.
	// +kubebuilder:validation:MinItems=0
	EventTypes []string `json:"eventTypes,omitempty"`

	// MatchingPatterns optional regex pattern that must be contained in the message to be logged
	// +kubebuilder:validation:MinItems=0
	MatchingPatterns []string `json:"matchingPatterns,omitempty"`

	// SkipOnMatch skip the entry if matched
	SkipOnMatch *bool `json:"skipOnMatch,omitempty"`
}

// LogField defines a log field
type LogField struct {
	// name of the log field
	Name string `json:"name"`
	// Path within the corev1.Event struct https://github.com/kubernetes/api/blob/master/core/v1/types.go
	// +kubebuilder:validation:MinItems=1
	Path []string `json:"path,omitempty"`
}

// EventLoggerStatus defines the observed state of EventLogger
type EventLoggerStatus struct {
	// OperatorVersion the version of the operator that processed the cr
	OperatorVersion string `json:"operatorVersion"`
	// LastProcessed the timestamp the cr was last processed
	LastProcessed metav1.Time `json:"lastProcessed"`

	// Error
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true

// EventLogger is the Schema for the eventloggers API
type EventLogger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventLoggerSpec   `json:"spec,omitempty"`
	Status EventLoggerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventLoggerList contains a list of EventLogger
type EventLoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventLogger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventLogger{}, &EventLoggerList{})
}

// Apply update the status of the current event logger
func (el *EventLogger) Apply(err error) {
	if err != nil {
		el.Status.Error = err.Error()
	} else {
		el.Status.Error = ""
	}
	el.Status.LastProcessed = metav1.Now()
	el.Status.OperatorVersion = version.Version

}
