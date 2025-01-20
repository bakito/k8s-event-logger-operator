package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V1", func() {
	var el *EventLogger
	var val *validateEl
	BeforeEach(func() {
		val = &validateEl{}
		el = &EventLogger{
			Spec: EventLoggerSpec{},
		}
	})
	Context("Valid", func() {
		Context("ValidateCreate", func() {
			It("should be valid", func() {
				w, err := val.ValidateCreate(context.TODO(), el)
				Ω(w).Should(BeNil())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be valid", func() {
				w, err := val.ValidateUpdate(context.TODO(), el, nil)
				Ω(w).Should(BeNil())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be nil", func() {
				w, err := val.ValidateDelete(context.TODO(), el)
				Ω(w).Should(BeNil())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})
	Context("Invalid", func() {
		BeforeEach(func() {
			el.Spec.Labels = map[string]string{"in valid": "valid"}
		})
		Context("ValidateCreate", func() {
			It("should be invalid", func() {
				w, err := val.ValidateCreate(context.TODO(), el)
				Ω(w).Should(BeNil())
				Ω(err).Should(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be invalid", func() {
				w, err := val.ValidateUpdate(context.TODO(), el, nil)
				Ω(w).Should(BeNil())
				Ω(err).Should(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be nil", func() {
				w, err := val.ValidateDelete(context.TODO(), el)
				Ω(w).Should(BeNil())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})
})
