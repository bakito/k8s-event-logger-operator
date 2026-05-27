package filter_test

import (
	corev1 "k8s.io/api/core/v1"

	f "github.com/bakito/k8s-event-logger-operator/pkg/filter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V1", func() {
	Context("Filter", func() {
		It("should match f.Always", func() {
			Ω(f.Always.Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Always.Match(nil)).Should(BeTrue())
			Ω(f.Always.String()).Should(Equal("true"))
		})
		It("should match f.Never", func() {
			Ω(f.Never.Match(&corev1.Event{})).Should(BeFalse())
			Ω(f.Never.Match(nil)).Should(BeFalse())
			Ω(f.Never.String()).Should(Equal("false"))
		})
		It("should match with func", func() {
			description := "type =='Bar'"
			filter := f.New(func(event *corev1.Event) bool {
				return event.Type == "Bar"
			}, description)

			Ω(filter.Match(&corev1.Event{Type: "Foo"})).Should(BeFalse())
			Ω(filter.Match(&corev1.Event{Type: "Bar"})).Should(BeTrue())
			Ω(filter.String()).Should(Equal(description))
		})
		It("should match all", func() {
			Ω(f.Slice{f.Always, f.Always, f.Always}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Slice{f.Always}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Slice{}.All().Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Slice{f.Never, f.Always}.All().Match(&corev1.Event{})).Should(BeFalse())
			Ω(f.Slice{f.Never, f.Always, f.Never}.All().String()).Should(Equal("( false AND true AND false )"))
		})
		It("should match any", func() {
			Ω(f.Slice{f.Never, f.Always, f.Never}.Any().Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Slice{f.Always}.Any().Match(&corev1.Event{})).Should(BeTrue())
			Ω(f.Slice{}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(f.Slice{f.Never}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(f.Slice{f.Never, f.Never}.Any().Match(&corev1.Event{})).Should(BeFalse())
			Ω(f.Slice{f.Never, f.Always, f.Never}.Any().String()).Should(Equal("( false OR true OR false )"))
		})
		It("should match nested f.Slice", func() {
			filter := f.Slice{f.Slice{f.Never, f.Always}.Any(), f.Slice{f.Always, f.Always}.All()}.All()
			Ω(filter.Match(&corev1.Event{})).Should(BeTrue())
			Ω(filter.String()).Should(Equal("( ( false OR true ) AND ( true AND true ) )"))
		})
		It("same filters should be equal", func() {
			filter1 := f.Slice{f.Slice{f.Never, f.Always}.Any(), f.Slice{f.Always, f.Always}.All()}.All()
			filter2 := f.Slice{f.Slice{f.Never, f.Always}.Any(), f.Slice{f.Always, f.Always}.All()}.All()
			Ω(filter1.Equals(filter2)).Should(BeTrue())
		})
		It("different filters should be equal", func() {
			filter1 := f.Slice{f.Slice{f.Never, f.Always}.Any(), f.Slice{f.Always, f.Never}.All()}.All()
			filter2 := f.Slice{f.Slice{f.Never, f.Always}.Any(), f.Slice{f.Always, f.Always}.All()}.All()
			Ω(filter1.Equals(filter2)).Should(BeFalse())
		})
	})
})
