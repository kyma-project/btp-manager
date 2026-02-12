package secrets

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var requiredSecretKeys = []string{"clientid", "clientsecret", "sm_url", "tokenurl", "cluster_id"}

type Verificator interface {
	Verify(secret *corev1.Secret) error
}

type VerificationDispatcher struct {
	verificators map[string]Verificator
}

func NewVerificationDispatcher(verificators map[string]Verificator) *VerificationDispatcher {
	if verificators == nil {
		verificators = make(map[string]Verificator)
	}
	return &VerificationDispatcher{
		verificators: verificators,
	}
}

func (d *VerificationDispatcher) Verify(secret *corev1.Secret) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	verificator, exists := d.verificators[secret.Name]
	if !exists {
		return fmt.Errorf("no verificator registered for secret: %s", secret.Name)
	}

	return verificator.Verify(secret)
}

func (d *VerificationDispatcher) RegisterVerificator(secretName string, verificator Verificator) {
	d.verificators[secretName] = verificator
}

type RequiredSecretVerificator struct {
	requiredKeys []string
}

func NewRequiredSecretVerificator() *RequiredSecretVerificator {
	return &RequiredSecretVerificator{requiredKeys: requiredSecretKeys}
}

func (v *RequiredSecretVerificator) Verify(secret *corev1.Secret) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	missingKeys := make([]string, 0)
	missingValues := make([]string, 0)
	errs := make([]string, 0)
	for _, key := range v.requiredKeys {
		value, exists := secret.Data[key]
		if !exists {
			missingKeys = append(missingKeys, key)
			continue
		}
		if len(value) == 0 {
			missingValues = append(missingValues, key)
		}
	}
	if len(missingKeys) > 0 {
		missingKeysMsg := fmt.Sprintf("key(s) %s not found", strings.Join(missingKeys, ", "))
		errs = append(errs, missingKeysMsg)
	}
	if len(missingValues) > 0 {
		missingValuesMsg := fmt.Sprintf("missing value(s) for %s key(s)", strings.Join(missingValues, ", "))
		errs = append(errs, missingValuesMsg)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}

type NoopVerificator struct{}

func NewNoopVerificator() *NoopVerificator {
	return &NoopVerificator{}
}

func (v *NoopVerificator) Verify(_ *corev1.Secret) error {
	return nil
}
