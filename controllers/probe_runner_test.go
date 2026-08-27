package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/btp-manager/controllers/config"
)

var _ = Describe("ProbeRunner", Label("probe-runner"), func() {
	var runner *ProbeRunner

	BeforeEach(func() {
		runner = &ProbeRunner{client: k8sClient}
	})

	AfterEach(func() {
		job := &batchv1.Job{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: probeJobName, Namespace: config.KymaSystemNamespaceName}, job)
		if err == nil {
			propagation := metav1.DeletePropagationBackground
			_ = k8sClient.Delete(ctx, job, &client.DeleteOptions{PropagationPolicy: &propagation})
		}
	})

	Describe("waitForJob", func() {
		It("times out internally when the job stays pending and no external deadline is set", func() {
			origTimeout := jobWaitTimeout
			jobWaitTimeout = 300 * time.Millisecond
			defer func() { jobWaitTimeout = origTimeout }()

			backoffLimit := int32(0)
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      probeJobName,
					Namespace: config.KymaSystemNamespaceName,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{Name: "probe", Image: "busybox:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			// Pass a background context with no deadline — waitForJob must impose its own.
			err := runner.waitForJob(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("deleteOldJob", func() {
		It("does not return AlreadyExists when createJob is called immediately after deleteOldJob", func() {
			origTimeout := jobWaitTimeout
			jobWaitTimeout = 5 * time.Second
			defer func() { jobWaitTimeout = origTimeout }()

			backoffLimit := int32(0)
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      probeJobName,
					Namespace: config.KymaSystemNamespaceName,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{Name: "probe", Image: "busybox:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			Expect(runner.deleteOldJob(ctx)).To(Succeed())
			runner.probeImage = "busybox:latest"
			Expect(runner.createJob(ctx)).To(Succeed())
		})
	})
})
