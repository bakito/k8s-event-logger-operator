package filter

import (
	"testing"

	. "gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_Filter_Match_Always(t *testing.T) {
	Assert(t, Always.Match(&corev1.Event{}) == true)
	Assert(t, Always.Match(nil) == true)
}

func Test_Filter_Match_Never(t *testing.T) {
	Assert(t, Always.Match(&corev1.Event{}) == false)
	Assert(t, Always.Match(nil) == false)
}

func Test_Filter_Match_Func(t *testing.T) {
	filter := New(func(event *corev1.Event) bool {
		return event.Type == "Bar"
	})

	Assert(t, filter.Match(&corev1.Event{Type: "Foo"}) == false)
	Assert(t, filter.Match(&corev1.Event{Type: "Bar"}) == true)
}

func Test_Slice_Match_All(t *testing.T) {
	Assert(t, Slice{Always, Always, Always}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Always}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Never, Always}.All().Match(&corev1.Event{}) == false)
}

func Test_Slice_Match_Any(t *testing.T) {
	Assert(t, Slice{Never, Always, Never}.Any().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Always}.Any().Match(&corev1.Event{}) == true)
	Assert(t, Slice{}.Any().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never}.Any().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never, Never}.Any().Match(&corev1.Event{}) == false)
}
