package v1_test

import (
	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V1", func() {
	var el *v1.EventLogger
	BeforeEach(func() {
		el = &v1.EventLogger{
			Spec: v1.EventLoggerSpec{},
		}
	})
	Context("Valid", func() {
		Context("ValidateCreate", func() {
			It("should be valid", func() {
				Ω(el.ValidateCreate()).ShouldNot(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be valid", func() {
				Ω(el.ValidateUpdate(nil)).ShouldNot(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be nil", func() {
				Ω(el.ValidateDelete()).Should(BeNil())
			})
		})
	})
	Context("Invalid", func() {
		BeforeEach(func() {
			el.Spec.Labels = map[string]string{"in valid": "valid"}
		})
		Context("ValidateCreate", func() {
			It("should be invalid", func() {
				Ω(el.ValidateCreate()).Should(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be invalid", func() {
				Ω(el.ValidateUpdate(nil)).Should(HaveOccurred())
			})
		})
		Context("ValidateUpdate", func() {
			It("should be nil", func() {
				Ω(el.ValidateDelete()).Should(BeNil())
			})
		})
	})
})
