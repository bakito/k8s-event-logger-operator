package v1_test

import (
	"encoding/json"
	"errors"

	v1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("V1", func() {
	Context("APIGroup serialisation", func() {
		It("should serialize an empty string", func() {
			k := &v1.Kind{
				Name:     "a",
				APIGroup: ptr.To(""),
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
				APIGroup: ptr.To("b"),
			}
			b, err := json.Marshal(k)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`{"name":"a","apiGroup":"b"}`))
		})
	})
	Context("Apply", func() {
		var el *v1.EventLogger
		BeforeEach(func() {
			el = &v1.EventLogger{}
		})
		It("should not set an error message", func() {
			el.Apply(nil)
			Ω(el.Status.Error).Should(BeEmpty())
			Ω(el.Status.OperatorVersion).Should(Equal(version.Version))
			Ω(el.Status.LastProcessed).ShouldNot(Equal(Equal(metav1.Time{})))
		})
		It("should set an error message", func() {
			err := errors.New("this is an error")
			el.Apply(err)
			Ω(el.Status.Error).Should(Equal(err.Error()))
			Ω(el.Status.OperatorVersion).Should(Equal(version.Version))
			Ω(el.Status.LastProcessed).ShouldNot(Equal(Equal(metav1.Time{})))
		})
	})
})
