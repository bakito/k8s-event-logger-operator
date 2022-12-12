package v1_test

import (
	"encoding/json"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

var _ = Describe("V1", func() {
	Context("APIGroup serialisation", func() {
		It("should serialize an empty string", func() {
			k := &v1.Kind{
				Name:     "a",
				APIGroup: pointer.String(""),
			}
			b, err := json.Marshal(k)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`{"name":"a","apiGroup":""}`))
		})
		It("not add apiGroups if nil", func() {
			k := &v1.Kind{
				Name: "a",
			}
			b, err := json.Marshal(k)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`{"name":"a"}`))
		})
		It("should serialize an the apiGroup value", func() {
			k := &v1.Kind{
				Name:     "a",
				APIGroup: pointer.String("b"),
			}
			b, err := json.Marshal(k)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`{"name":"a","apiGroup":"b"}`))
		})
	})
})
