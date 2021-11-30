package filter_test

import (
	. "github.com/bakito/k8s-event-logger-operator/pkg/filter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("V1", func() {
	Context("Filter", func() {
		It("should match always", func() {
			Ω(Always.Match(&corev1.Event{})).Should(BeTrue())
			Ω(Always.Match(nil)).Should(BeTrue())
			Ω(Always.String()).Should(Equal("true"))
		})
		It("should match never", func() {
			Ω(Never.Match(&corev1.Event{})).Should(BeFalse())
			Ω(Never.Match(nil)).Should(BeFalse())
			Ω(Never.String()).Should(Equal("false"))
		})
		It("should match with func", func() {
			description := "type =='Bar'"
			filter := New(func(event *corev1.Event) bool {
				return event.Type == "Bar"
			}, description)

			Ω(filter.Match(&corev1.Event{Type: "Foo"})).Should(BeFalse())
			Ω(filter.Match(&corev1.Event{Type: "Bar"})).Should(BeTrue())
			Ω(filter.String()).Should(Equal(description))
		})
		It("should match all", func() {
			Ω(Slice{Always, Always, Always}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(Slice{Always}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(Slice{}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(Slice{Never, Always}.All().Match(&corev1.Event{})).Should(BeFalse())
			Ω(Slice{Never, Always, Never}.All().String()).Should(Equal("( false AND true AND false )"))
		})
		It("should match any", func() {
			Ω(Slice{Never, Always, Never}.Any().Match(&corev1.Event{})).Should(BeTrue())
			Ω(Slice{Always}.Any().Match(&corev1.Event{})).Should(BeTrue())
			Ω(Slice{}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(Slice{Never}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(Slice{Never, Never}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(Slice{Never, Always, Never}.Any().String()).Should(Equal("( false OR true OR false )"))
		})
		It("should match nested slice", func() {
			filter := Slice{Slice{Never, Always}.Any(), Slice{Always, Always}.All()}.All()
			Ω(filter.Match(&corev1.Event{})).Should(BeTrue())
			Ω(filter.String()).Should(Equal("( ( false OR true ) AND ( true AND true ) )"))
		})
	})
})
