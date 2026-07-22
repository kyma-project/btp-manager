package configurator

import (
	"context"
	"fmt"

	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CheckResult tells the reconciler what state transition (if any) to make.
// All fields empty means no action needed.
type CheckResult struct {
	ReprocessReason  conditions.Reason
	ReprocessMessage string
	ErrorReason      conditions.Reason
	ErrorMessage     string
}

type SapBtpServiceOperatorConfigurator interface {
	Check(ctx context.Context) CheckResult
}

type configurator struct {
	driftDetector drift.Detector
}

func NewConfigurator(d drift.Detector) SapBtpServiceOperatorConfigurator {
	return &configurator{driftDetector: d}
}

var _ SapBtpServiceOperatorConfigurator = (*configurator)(nil)

func (c *configurator) Check(ctx context.Context) CheckResult {
	logger := log.FromContext(ctx)

	defaultCredentialsSecret, err := c.driftDetector.GetDefaultCredentialsSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting default credentials secret")
		return CheckResult{
			ErrorReason:  conditions.GettingDefaultCredentialsSecretFailed,
			ErrorMessage: err.Error(),
		}
	}
	if defaultCredentialsSecret != nil {
		managerNs := c.driftDetector.CredentialsNamespaceFromManager()
		if managerNs != defaultCredentialsSecret.Namespace {
			msg := fmt.Sprintf("credentials namespace changed from %s to %s", defaultCredentialsSecret.Namespace, managerNs)
			logger.Info(msg)
			return CheckResult{ReprocessReason: conditions.CredentialsNamespaceChanged, ReprocessMessage: msg}
		}
	}

	sapBtpOperatorConfigMap, err := c.driftDetector.GetSapBtpServiceOperatorConfigMap(ctx)
	if err != nil {
		logger.Error(err, "while getting sap-btp-operator config map")
		return CheckResult{
			ErrorReason:  conditions.GettingSapBtpServiceOperatorConfigMapFailed,
			ErrorMessage: err.Error(),
		}
	}
	if sapBtpOperatorConfigMap != nil {
		clusterIdFromCM := sapBtpOperatorConfigMap.Data[drift.ClusterIdConfigMapKey]
		if c.driftDetector.ClusterIdFromManager() != clusterIdFromCM {
			msg := fmt.Sprintf("cluster ID changed from %s to %s", clusterIdFromCM, c.driftDetector.ClusterIdFromManager())
			logger.Info(msg)
			return CheckResult{ReprocessReason: conditions.ClusterIdChanged, ReprocessMessage: msg}
		}
	}

	return CheckResult{}
}
