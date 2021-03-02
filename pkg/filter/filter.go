package filter

import (
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
}

// New creates a new Filter from a filter function: func(*corev1.Event) bool
func New(f func(*corev1.Event) bool) Filter {
	return &Func{Func: f}
}

// Func is a generic Filter
type Func struct {
	Func func(*corev1.Event) bool
}

// Match implements Filter interface
func (f *Func) Match(e *corev1.Event) bool {
	return f.Func(e)
}

// Equals implements Filter interface
func (f *Func) Equals(o Filter) bool {
	return cmp.Equal(f, o, cmpopts.EquateEmpty())
}

// Never is a filter that never matches
var Never = &Func{
	Func: func(_ *corev1.Event) bool {
		return false
	},
}

// Always is a filter that always matches
var Always = &Func{
	Func: func(_ *corev1.Event) bool {
		return true
	},
}

// Slice is a slice of Filter
type Slice []Filter

// Any creates a new Filter which checks if least one Filter in the Slice matches (if the Slice is empty this is equivalent to Never)
func (s Slice) Any() Filter {
	return &Func{
		func(e *corev1.Event) bool {
			for _, filter := range s {
				if filter.Match(e) {
					return true
				}
			}
			return false
		},
	}
}

// All creates a new Filter which checks if all Filter in the Slice matches (if the Slice is empty this is equivalent to Always)
func (s Slice) All() Filter {
	return &Func{
		func(e *corev1.Event) bool {
			for _, filter := range s {
				if !filter.Match(e) {
					return false
				}
			}
			return true
		},
	}
}
