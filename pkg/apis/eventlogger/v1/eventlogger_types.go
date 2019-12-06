package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventLoggerConf defines the configuration of EventLogger
// +k8s:openapi-gen=true
type EventLoggerConf struct {

	// Kinds the kinds to logg the events for
	// +kubebuilder:validation:MinItems=1
	// +listType=set
	Kinds []Kind `json:"kinds,omitempty"`

	// EventTypes the event types to log. If empty all events are logged.
	// +kubebuilder:validation:MinItems=0
	// +listType=set
	EventTypes []string `json:"eventTypes,omitempty"`
}

// EventLoggerSpec defines the desired state of EventLogger
// +k8s:openapi-gen=true
type EventLoggerSpec struct {
	EventLoggerConf `json:",inline"`

	// Labels additional labels for the logger pod
	Labels map[string]string `json:"labels,omitempty"`

	// Labels additional annotations for the logger pod
	Annotations map[string]string `json:"annotations,omitempty"`

	// ScrapeMetrics if true, prometheus scrape annotations are added to the pod
	ScrapeMetrics *bool `json:"scrapeMetrics,omitempty"`

	// Namespace the namespace to watch on, may be an empty string
	// +nullable
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// ServiceAccount the service account to use for the logger pod
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// Kind defines a kind to loge events for
// +k8s:openapi-gen=true
type Kind struct {
	// +kubebuilder:validation:MinLength=3
	Name string `json:"name"`

	// EventTypes the event types to log. If empty events are logged as defined in spec.
	// +kubebuilder:validation:MinItems=0
	// +listType=set
	EventTypes []string `json:"eventTypes,omitempty"`

	// MatchingPatterns optional regex pattern that must be contained in the message to be logged
	// +kubebuilder:validation:MinItems=0
	// +listType=set
	MatchingPatterns []string `json:"matchingPatterns,omitempty"`

	// SkipOnMatch skip the entry if matched
	SkipOnMatch *bool `json:"skipOnMatch,omitempty"`
}

// EventLoggerStatus defines the observed state of EventLogger
// +k8s:openapi-gen=true
type EventLoggerStatus struct {
	// OperatorVersion the version of the operator that processed the cr
	OperatorVersion string `json:"operatorVersion"`
	// LastProcessed the timestamp the cr was last processed
	LastProcessed metav1.Time `json:"lastProcessed"`

	// Error
	Error string `json:"error,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventLogger is the Schema for the eventloggers API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=eventloggers,scope=Namespaced
type EventLogger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventLoggerSpec   `json:"spec"`
	Status EventLoggerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventLoggerList contains a list of EventLogger
type EventLoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventLogger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventLogger{}, &EventLoggerList{})
}
