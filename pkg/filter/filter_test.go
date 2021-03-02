package filter

import (
	"testing"

	. "gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_Filter_Match_Always(t *testing.T) {
	Assert(t, Always.Match(&corev1.Event{}) == true)
	Assert(t, Always.Match(nil) == true)
	Assert(t, Always.String() == "true")
}

func Test_Filter_Match_Never(t *testing.T) {
	Assert(t, Never.Match(&corev1.Event{}) == false)
	Assert(t, Never.Match(nil) == false)
	Assert(t, Never.String() == "false")
}

func Test_Filter_Match_Func(t *testing.T) {
	description := "type =='Bar'"
	filter := New(func(event *corev1.Event) bool {
		return event.Type == "Bar"
	}, description)

	Assert(t, filter.Match(&corev1.Event{Type: "Foo"}) == false)
	Assert(t, filter.Match(&corev1.Event{Type: "Bar"}) == true)
	Assert(t, filter.String() == description)
}

func Test_Slice_Match_All(t *testing.T) {
	Assert(t, Slice{Always, Always, Always}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Always}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{}.All().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Never, Always}.All().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never, Always, Never}.All().String() == "( false AND true AND false )")
}

func Test_Slice_Match_Any(t *testing.T) {
	Assert(t, Slice{Never, Always, Never}.Any().Match(&corev1.Event{}) == true)
	Assert(t, Slice{Always}.Any().Match(&corev1.Event{}) == true)
	Assert(t, Slice{}.Any().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never}.Any().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never, Never}.Any().Match(&corev1.Event{}) == false)
	Assert(t, Slice{Never, Always, Never}.Any().String() == "( false OR true OR false )")
}

func Test_Nested_Slice_Match(t *testing.T) {
	filter := Slice{Slice{Never, Always}.Any(), Slice{Always, Always}.All()}.All()
	Assert(t, filter.Match(&corev1.Event{}) == true)
	Assert(t, filter.String() == "( ( false OR true ) AND ( true AND true ) )")
}
