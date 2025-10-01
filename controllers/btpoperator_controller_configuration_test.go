package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BTP Operator controller - configuration", func() {
	Context("When the ConfigMap is present", func() {
		It("should adjust configuration settings in the operator accordingly", func() {
			GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
			cm := initConfig(map[string]string{"ProcessingStateRequeueInterval": "10s"})
			reconciler.reconcileConfig(context.TODO(), cm)
			Expect(ProcessingStateRequeueInterval).To(Equal(time.Second * 10))
		})

		Context("when EnableLimitedCache is configured", func() {
			var originalValue string

			BeforeEach(func() {
				originalValue = EnableLimitedCache
			})

			AfterEach(func() {
				EnableLimitedCache = originalValue
			})

			It("should set EnableLimitedCache to true when configured", func() {
				GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")

				cm := initConfig(map[string]string{"EnableLimitedCache": "true"})
				reconciler.reconcileConfig(context.TODO(), cm)
				Expect(EnableLimitedCache).To(Equal("true"))
			})

			It("should set EnableLimitedCache to false when configured", func() {
				GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")

				cm := initConfig(map[string]string{"EnableLimitedCache": "false"})
				reconciler.reconcileConfig(context.TODO(), cm)
				Expect(EnableLimitedCache).To(Equal("false"))
			})
		})
	})
})
