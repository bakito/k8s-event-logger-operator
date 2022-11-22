package v1_test

import (
	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V1", func() {
	Context("Validate", func() {
		It("should succeed", func() {
			s := &v1.EventLoggerSpec{
				Annotations: map[string]string{"valid/valid": "valid", "valid": "valid"},
				Labels:      map[string]string{"valid": "valid"},
			}
			Ω(s.Validate()).ShouldNot(HaveOccurred())
		})
		It("should have invalid label key", func() {
			s := &v1.EventLoggerSpec{
				Labels: map[string]string{"in valid": "valid"},
			}
			Ω(s.Validate()).Should(HaveOccurred())
			s = &v1.EventLoggerSpec{
				Labels: map[string]string{"in:valid": "valid"},
			}
			Ω(s.Validate()).Should(HaveOccurred())
		})
		It("should have invalid label value", func() {
			s := &v1.EventLoggerSpec{
				Labels: map[string]string{"valid": "in valid"},
			}
			Ω(s.Validate()).Should(HaveOccurred())
			s = &v1.EventLoggerSpec{
				Labels: map[string]string{"valid": "in:valid"},
			}
			Ω(s.Validate()).Should(HaveOccurred())
		})
		It("should have invalid annotation key", func() {
			s := &v1.EventLoggerSpec{
				Annotations: map[string]string{"in valid": "valid"},
			}
			s = &v1.EventLoggerSpec{
				Annotations: map[string]string{"in:valid:": "valid"},
			}
			Ω(s.Validate()).Should(HaveOccurred())
		})
	})
})
