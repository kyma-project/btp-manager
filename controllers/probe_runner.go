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
	probeAnnotationStatus    = "tls-probe-status"
	probeAnnotationHash      = "tls-probe-hash"
	probeAnnotationLastHash  = "tls-probe-last-hash"
	probeAnnotationUpdatedAt = "tls-probe-updated-at"

	probeStatusOK    = "ok"
	probeStatusAlert = "alert"
)

var jobWaitTimeout = 5 * time.Minute

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
		Help:      "CA bundle probe status: 0=ok or error, 1=alert (CA mounted but cert not trusted)",
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

	// Record tls-probe-updated-at before running the job so we can detect whether the probe
	// wrote new annotations this cycle (it silently exits without writing when no mount + TLS ok).
	prevUpdatedAt, err := r.getUpdatedAt(ctx)
	if err != nil {
		return fmt.Errorf("reading tls-probe-updated-at: %w", err)
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

	status := annotations[probeAnnotationStatus]
	hash := annotations[probeAnnotationHash]
	lastHash := annotations[probeAnnotationLastHash]
	updatedAt := annotations[probeAnnotationUpdatedAt]

	logger.Info("probe cycle result",
		"status", status,
		"hash", hash,
		"lastHash", lastHash,
		"updatedAt", updatedAt,
	)

	// If tls-probe-updated-at did not advance, the probe silently exited (no mount + TLS ok).
	// Nothing to process this cycle.
	if updatedAt == prevUpdatedAt {
		return nil
	}

	// Update metric: 1 if alert (CA mounted but cert not trusted — actionable), 0 otherwise.
	// error signals (connectivity failures, no mount) are logged but do not fire the metric.
	if status == probeStatusAlert {
		r.statusGauge.Set(1)
	} else {
		r.statusGauge.Set(0)
	}

	// Restart btp-operator pods when: TLS ok (status=ok), hash changed, and lastHash was non-empty (not first run).
	if status == probeStatusOK && hash != lastHash && lastHash != "" {
		logger.Info("CA bundle hash changed with healthy TLS — restarting btp-operator pods")
		if err := r.restartBtpOperatorPods(ctx); err != nil {
			logger.Error(err, "failed to restart btp-operator pods")
		}
	}

	// Advance tls-probe-last-hash only when probe wrote a non-empty hash.
	// Overwriting with empty would erase the previous hash and suppress future restart detection.
	if hash == "" {
		return nil
	}
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

func (r *ProbeRunner) getUpdatedAt(ctx context.Context) (string, error) {
	cr := &v1alpha1.BtpOperator{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Name:      config.BtpOperatorCrName,
		Namespace: config.KymaSystemNamespaceName,
	}, cr); err != nil {
		return "", fmt.Errorf("getting BtpOperator CR: %w", err)
	}
	return cr.GetAnnotations()[probeAnnotationUpdatedAt], nil
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
	if err := r.client.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil {
		return err
	}
	// Poll until the object is fully removed so the subsequent createJob call does not
	// receive an "object is being deleted" AlreadyExists error.
	deleteCtx, cancel := context.WithTimeout(ctx, jobWaitTimeout)
	defer cancel()
	for {
		select {
		case <-deleteCtx.Done():
			return deleteCtx.Err()
		default:
		}
		err := r.client.Get(deleteCtx, types.NamespacedName{
			Name:      probeJobName,
			Namespace: config.KymaSystemNamespaceName,
		}, &batchv1.Job{})
		if k8serrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
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
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: probeJobName,
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
	waitCtx, cancel := context.WithTimeout(ctx, jobWaitTimeout)
	defer cancel()
	// Poll until the job completes (succeeded or failed) or the deadline is exceeded.
	for {
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		default:
		}

		job := &batchv1.Job{}
		if err := r.client.Get(waitCtx, types.NamespacedName{
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
