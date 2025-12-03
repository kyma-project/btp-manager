package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configuration controller", func() {

	Context("When EnableLimitedCache is created/updated", func() {
		var originalValue string

		BeforeEach(func() {
			originalValue = EnableLimitedCache
		})

		AfterEach(func() {
			EnableLimitedCache = originalValue
		})

		It("should update EnableLimitedCache", func() {
			createOrUpdateConfigMap(map[string]string{
				"EnableLimitedCache": "true",
			})

			Eventually(func() string {
				return EnableLimitedCache
			}).Should(Equal("true"))
		})
	})

	Context("When ProcessingStateRequeueInterval is created/updated", func() {
		var originalValue time.Duration

		BeforeEach(func() {
			originalValue = ProcessingStateRequeueInterval
		})

		AfterEach(func() {
			ProcessingStateRequeueInterval = originalValue
		})

		It("should update ProcessingStateRequeueInterval", func() {
			createOrUpdateConfigMap(map[string]string{
				"ProcessingStateRequeueInterval": "10s",
			})

			Eventually(func() time.Duration {
				return ProcessingStateRequeueInterval
			}).Should(Equal(10 * time.Second))
		})
	})
})
