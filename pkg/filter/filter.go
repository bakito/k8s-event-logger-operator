package filter

import (
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
)

// Filter is a event filters
type Filter interface {
	// Match checks if a Event matches the filter
	Match(*corev1.Event) bool
	// Equals compares the Filter with another
	Equals(Filter) bool
	// String returns the description of the Filter
	String() string
}

// New creates a new Filter from a filter function: func(*corev1.Event) bool
func New(f func(*corev1.Event) bool, description string) Filter {
	return &Func{Func: f, Description: description}
}

// Func is a generic Filter
type Func struct {
	Func        func(*corev1.Event) bool
	Description string
}

// Match implements Filter interface
func (f *Func) Match(e *corev1.Event) bool {
	return f.Func(e)
}

// Equals implements Filter interface
func (f *Func) Equals(o Filter) bool {
	return cmp.Equal(f, o, cmpopts.EquateEmpty())
}

func (f *Func) String() string {
	return f.Description
}

// Never is a filter that never matches
var Never = &Func{
	Func: func(_ *corev1.Event) bool {
		return false
	},
	Description: "false",
}

// Always is a filter that always matches
var Always = &Func{
	Func: func(_ *corev1.Event) bool {
		return true
	},
	Description: "true",
}

// Slice is a slice of Filter
type Slice []Filter

// Any creates a new Filter which checks if least one Filter in the Slice matches (if the Slice is empty this is equivalent to Never)
func (s Slice) Any() Filter {
	return &Func{
		Func: func(e *corev1.Event) bool {
			for _, filter := range s {
				if filter.Match(e) {
					return true
				}
			}
			return false
		},
		Description: "( " + strings.Join(s.toStringSlice(), " OR ") + " )",
	}
}

// All creates a new Filter which checks if all Filter in the Slice matches (if the Slice is empty this is equivalent to Always)
func (s Slice) All() Filter {
	return &Func{
		Func: func(e *corev1.Event) bool {
			for _, filter := range s {
				if !filter.Match(e) {
					return false
				}
			}
			return true
		},
		Description: "( " + strings.Join(s.toStringSlice(), " AND ") + " )",
	}
}

// toStringSlice creates a slice with the descriptions of all the Filter in Slice
func (s Slice) toStringSlice() []string {
	var descriptions []string

	for _, filter := range s {
		descriptions = append(descriptions, filter.String())
	}

	return descriptions
}
