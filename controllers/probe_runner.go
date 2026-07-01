package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

//+kubebuilder:rbac:groups="batch",resources="jobs",verbs=get;create;delete

const (
	probeJobName             = "ca-bundle-probe"
	probeAnnotationTLSResult = "tls-probe-tls-result"
	probeAnnotationHash      = "tls-probe-hash"
	probeAnnotationLastHash  = "tls-probe-last-hash"
	probeAnnotationSignal    = "tls-probe-signal"

	probeSignalAlert = "alert"
	probeTLSResultOK = "ok"
)

// ProbeRunner is a controller-runtime Runnable that periodically spawns a tls-probe Job
// and reads back signals from BtpOperator CR annotations. It is disabled when ProbeInterval is 0.
type ProbeRunner struct {
	client           client.Client
	probeInterval    time.Duration
	probeImage       string
	tokenURLOverride string
	statusGauge      prometheus.Gauge
}

func NewProbeRunner(c client.Client, registry prometheus.Registerer) *ProbeRunner {
	interval := parseProbeInterval(os.Getenv("PROBE_INTERVAL"))
	image := os.Getenv("PROBE_IMAGE")
	override := os.Getenv("PROBE_TOKENURL_OVERRIDE")

	gauge := promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Namespace: "btpmanager",
		Name:      "credential_probe_status",
		Help:      "CA bundle probe status: 0=healthy, 1=unhealthy (alert-level signal)",
	})

	return &ProbeRunner{
		client:           c,
		probeInterval:    interval,
		probeImage:       image,
		tokenURLOverride: override,
		statusGauge:      gauge,
	}
}

func parseProbeInterval(raw string) time.Duration {
	if raw == "" || raw == "0" || raw == "0s" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// Start implements manager.Runnable. Returns immediately if probe is disabled.
//
//nolint:cyclop
func (r *ProbeRunner) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("probe-runner")

	if r.probeInterval == 0 || r.probeImage == "" {
		logger.Info("CA bundle probe disabled", "interval", r.probeInterval, "image", r.probeImage)
		return nil
	}

	logger.Info("CA bundle probe runner started", "interval", r.probeInterval, "image", r.probeImage)

	ticker := time.NewTicker(r.probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.runCycle(ctx); err != nil {
				logger.Error(err, "probe cycle failed")
			}
		}
	}
}

func (r *ProbeRunner) runCycle(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("probe-runner")

	if err := r.deleteOldJob(ctx); err != nil {
		return fmt.Errorf("deleting old probe job: %w", err)
	}

	if err := r.createJob(ctx); err != nil {
		return fmt.Errorf("creating probe job: %w", err)
	}

	if err := r.waitForJob(ctx); err != nil {
		return fmt.Errorf("waiting for probe job: %w", err)
	}

	cr := &v1alpha1.BtpOperator{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Name:      config.BtpOperatorCrName,
		Namespace: config.KymaSystemNamespaceName,
	}, cr); err != nil {
		return fmt.Errorf("getting BtpOperator CR: %w", err)
	}

	annotations := cr.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	signal := annotations[probeAnnotationSignal]
	tlsResult := annotations[probeAnnotationTLSResult]
	hash := annotations[probeAnnotationHash]
	lastHash := annotations[probeAnnotationLastHash]

	logger.Info("probe cycle result",
		"signal", signal,
		"tls-result", tlsResult,
		"hash", hash,
		"lastHash", lastHash,
	)

	// Update metric: 1 if alert, 0 otherwise.
	if signal == probeSignalAlert {
		r.statusGauge.Set(1)
	} else {
		r.statusGauge.Set(0)
	}

	// Restart btp-operator pods when: hash changed, TLS ok, and lastHash was non-empty (not first run).
	if tlsResult == probeTLSResultOK && hash != lastHash && lastHash != "" {
		logger.Info("CA bundle hash changed with healthy TLS — restarting btp-operator pods")
		if err := r.restartBtpOperatorPods(ctx); err != nil {
			logger.Error(err, "failed to restart btp-operator pods")
		}
	}

	// Advance tls-probe-last-hash after processing.
	patch := client.MergeFrom(cr.DeepCopy())
	if cr.Annotations == nil {
		cr.Annotations = map[string]string{}
	}
	cr.Annotations[probeAnnotationLastHash] = hash
	if err := r.client.Patch(ctx, cr, patch); err != nil {
		return fmt.Errorf("patching tls-probe-last-hash: %w", err)
	}

	return nil
}

func (r *ProbeRunner) deleteOldJob(ctx context.Context) error {
	job := &batchv1.Job{}
	err := r.client.Get(ctx, types.NamespacedName{
		Name:      probeJobName,
		Namespace: config.KymaSystemNamespaceName,
	}, job)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	propagation := metav1.DeletePropagationBackground
	return r.client.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

func (r *ProbeRunner) createJob(ctx context.Context) error {
	backoffLimit := int32(0)
	env := []corev1.EnvVar{
		{Name: "PROBE_NAMESPACE", Value: config.KymaSystemNamespaceName},
	}
	if r.tokenURLOverride != "" {
		env = append(env, corev1.EnvVar{Name: "PROBE_TOKENURL_OVERRIDE", Value: r.tokenURLOverride})
	}

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
						{
							Name:  "probe",
							Image: r.probeImage,
							Env:   env,
						},
					},
				},
			},
		},
	}
	return r.client.Create(ctx, job)
}

func (r *ProbeRunner) waitForJob(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("probe-runner")
	// Poll until the job completes (succeeded or failed) or context is cancelled.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job := &batchv1.Job{}
		if err := r.client.Get(ctx, types.NamespacedName{
			Name:      probeJobName,
			Namespace: config.KymaSystemNamespaceName,
		}, job); err != nil {
			return err
		}

		if job.Status.Succeeded > 0 {
			return nil
		}
		if job.Status.Failed > 0 {
			logger.Info("probe job failed; reading annotations anyway")
			return nil
		}

		time.Sleep(2 * time.Second)
	}
}

func (r *ProbeRunner) restartBtpOperatorPods(ctx context.Context) error {
	podList := &corev1.PodList{}
	if err := r.client.List(ctx, podList,
		client.InNamespace(config.KymaSystemNamespaceName),
		client.MatchingLabels{"control-plane": "controller-manager"},
	); err != nil {
		return err
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if err := r.client.Delete(ctx, pod); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
