package logging

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logging", func() {
	Context("ConfigFor", func() {
		It("should create a correct config", func() {
			name := uuid.NewString()
			podNs := uuid.NewString()
			watchNs := uuid.NewString()
			cfg := ConfigFor(name, podNs, watchNs)
			Ω(cfg.name).Should(Equal(name))
			Ω(cfg.podNamespace).Should(Equal(podNs))
			Ω(cfg.watchNamespace).Should(Equal(watchNs))
		})
	})
})
