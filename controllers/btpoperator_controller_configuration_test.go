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
	})
})
