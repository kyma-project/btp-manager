package secrets

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var requiredSecretKeys = []string{"clientid", "clientsecret", "sm_url", "tokenurl", "cluster_id"}

type Verifier interface {
	Verify(secret *corev1.Secret) error
}

type VerificationDispatcher struct {
	verifiers map[string]Verifier
}

func NewVerificationDispatcher(verifiers map[string]Verifier) *VerificationDispatcher {
	if verifiers == nil {
		verifiers = make(map[string]Verifier)
	}
	return &VerificationDispatcher{
		verifiers: verifiers,
	}
}

func (d *VerificationDispatcher) Verify(secret *corev1.Secret) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	verifier, exists := d.verifiers[secret.Name]
	if !exists {
		return fmt.Errorf("no verifier registered for secret: %s", secret.Name)
	}

	return verifier.Verify(secret)
}

func (d *VerificationDispatcher) RegisterVerifier(secretName string, verifier Verifier) {
	d.verifiers[secretName] = verifier
}

type RequiredSecretVerifier struct {
	requiredKeys []string
}

func NewRequiredSecretVerifier() *RequiredSecretVerifier {
	return &RequiredSecretVerifier{requiredKeys: requiredSecretKeys}
}

func (v *RequiredSecretVerifier) Verify(secret *corev1.Secret) error {
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

type NoopVerifier struct{}

func NewNoopVerifier() *NoopVerifier {
	return &NoopVerifier{}
}

func (v *NoopVerifier) Verify(_ *corev1.Secret) error {
	return nil
}
