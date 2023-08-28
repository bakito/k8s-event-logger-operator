package logging

import (
	"fmt"
	"regexp"
	"strings"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/pkg/filter"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func newFilter(c eventloggerv1.EventLoggerSpec) filter.Filter {
	filters := filter.Slice{}

	if len(c.EventTypes) > 0 {
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return contains(c.EventTypes, e.Type)
		}, fmt.Sprintf("EventType in [%s]", strings.Join(c.EventTypes, ", "))))
	}

	if len(c.Kinds) > 0 {
		filterForKinds := filter.Slice{}
		for _, k := range c.Kinds {
			if len(k.EventTypes) == 0 {
				k.EventTypes = c.EventTypes
			}

			filterForKinds = append(filterForKinds, newFilterForKind(k))
		}

		filters = append(filters, filterForKinds.Any())
	}

	if len(filters) == 0 {
		return filter.Always
	}

	return filters.Any()
}

func newFilterForKind(k eventloggerv1.Kind) filter.Filter {
	filters := filter.Slice{}

	filters = append(filters, filter.New(func(e *corev1.Event) bool {
		return k.Name == e.InvolvedObject.Kind
	}, fmt.Sprintf("Kind == '%s'", k.Name)))

	if k.APIGroup != nil {
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return *k.APIGroup == e.InvolvedObject.GroupVersionKind().Group
		}, fmt.Sprintf("APIGroup == '%s'", *k.APIGroup)))
	}

	if len(k.EventTypes) > 0 {
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return contains(k.EventTypes, e.Type)
		}, fmt.Sprintf("EventType in [%s]", strings.Join(k.EventTypes, ", "))))
	}

	if len(k.SkipReasons) > 0 {
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return !contains(k.SkipReasons, e.Reason)
		}, fmt.Sprintf("Reason NOT in [%s]", strings.Join(k.SkipReasons, ", "))))
	}

	if len(k.Reasons) > 0 {
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return contains(k.Reasons, e.Reason)
		}, fmt.Sprintf("Reason in [%s]", strings.Join(k.Reasons, ", "))))
	}

	if k.MatchingPatterns != nil {
		filters = append(filters, newFilterForMatchingPatterns(k.MatchingPatterns, ptr.Deref(k.SkipOnMatch, false)))
	}

	return filters.All()
}

func newFilterForMatchingPatterns(patterns []string, skipOnMatch bool) filter.Filter {
	filters := filter.Slice{}
	for _, mp := range patterns {
		matcher := regexp.MustCompile(mp)
		filters = append(filters, filter.New(func(e *corev1.Event) bool {
			return matcher.Match([]byte(e.Message))
		}, fmt.Sprintf("Message matches /%s/", mp)))
	}

	f := filters.Any()
	return filter.New(func(e *corev1.Event) bool {
		return skipOnMatch != f.Match(e)
	}, fmt.Sprintf("( %v XOR %s )", skipOnMatch, f.String()))
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
	filter         filter.Filter
}

func (c Config) matches(meta metav1.Object) bool {
	if c.watchNamespace == "" {
		return c.podNamespace == meta.GetNamespace() && (c.name == meta.GetName())
	}
	return c.watchNamespace == meta.GetNamespace() && (c.name == meta.GetName())
}

// contains check if a string in a []string exists
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
