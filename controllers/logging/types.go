package logging

import (
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// KindFilter filter for kind
type KindFilter struct {
	EventTypes       []string         `json:"eventTypes,omitempty"`
	MatchingPatterns []*regexp.Regexp `json:"matchingPatterns,omitempty"`
	SkipOnMatch      bool             `json:"skipOnMatch,omitempty"`
}

// Filter event filter
type Filter struct {
	Kinds      map[string]*KindFilter `json:"kinds,omitempty"`
	EventTypes []string               `json:"eventTypes,omitempty"`
}

// Equals check if the filter equals the other
func (f *Filter) Equals(o *Filter) bool {
	return cmp.Equal(f, o, cmpopts.EquateEmpty())
}

func newFilter(c eventloggerv1.EventLoggerSpec) *Filter {
	f := &Filter{}
	f.EventTypes = c.EventTypes
	f.Kinds = make(map[string]*KindFilter)
	for _, k := range c.Kinds {
		kp := &k
		f.Kinds[k.Name] = &KindFilter{
			MatchingPatterns: []*regexp.Regexp{},
		}
		if kp.EventTypes == nil {
			f.Kinds[k.Name].EventTypes = c.EventTypes
		} else {
			f.Kinds[k.Name].EventTypes = kp.EventTypes
		}

		if k.MatchingPatterns != nil {
			f.Kinds[k.Name].SkipOnMatch = k.SkipOnMatch != nil && *k.SkipOnMatch
			for _, mp := range k.MatchingPatterns {
				f.Kinds[k.Name].MatchingPatterns = append(f.Kinds[k.Name].MatchingPatterns, regexp.MustCompile(mp))
			}
		}
	}
	return f
}

// ConfigFor get config for namespace and name
func ConfigFor(name, podNamespace, watchNamespace string) *Config {
	return &Config{
		name:           name,
		podNamespace:   podNamespace,
		watchNamespace: watchNamespace,
	}
}

// Config event config
type Config struct {
	podNamespace   string
	watchNamespace string
	name           string
	logFields      []eventloggerv1.LogField
	filter         *Filter
}

func (c Config) matches(meta metav1.Object) bool {
	if c.watchNamespace == "" {
		return c.podNamespace == meta.GetNamespace() && (c.name == meta.GetName())
	}
	return c.watchNamespace == meta.GetNamespace() && (c.name == meta.GetName())
}
