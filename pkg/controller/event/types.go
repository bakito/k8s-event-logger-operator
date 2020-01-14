package event

import (
	"regexp"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// KindFilter filter for kind
type KindFilter struct {
	eventTypes       []string
	matchingPatterns []*regexp.Regexp
	skipOnMatch      bool
}

// Filter event filter
type Filter struct {
	Kinds      map[string]*KindFilter
	EventTypes []string
}

// Equals check if the filter equals the other
func (f *Filter) Equals(o *Filter) bool {
	return cmp.Equal(f, o, cmpopts.EquateEmpty())
}

func newFilter(c eventloggerv1.EventLoggerConf) *Filter {
	f := &Filter{}
	f.EventTypes = c.EventTypes
	f.Kinds = make(map[string]*KindFilter)
	for _, k := range c.Kinds {
		kp := &k
		f.Kinds[k.Name] = &KindFilter{
			matchingPatterns: []*regexp.Regexp{},
		}
		if kp.EventTypes == nil {
			f.Kinds[k.Name].eventTypes = c.EventTypes
		} else {
			f.Kinds[k.Name].eventTypes = kp.EventTypes
		}

		if k.MatchingPatterns != nil {
			f.Kinds[k.Name].skipOnMatch = k.SkipOnMatch != nil && *k.SkipOnMatch
			for _, mp := range k.MatchingPatterns {
				f.Kinds[k.Name].matchingPatterns = append(f.Kinds[k.Name].matchingPatterns, regexp.MustCompile(mp))
			}
		}
	}
	return f
}
