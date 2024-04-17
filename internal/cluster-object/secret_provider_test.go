package clusterobject

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/btp-manager/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretProvider(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("should fetch all secrets - from the module's namespace, with a namespace prefix, with an arbitrary name", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		additionalNamespaces := createAdditionalNamespaces()
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      btpServiceOperatorSecretName,
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      btpServiceOperatorSecretName,
					Namespace: "test1",
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test2-%s", btpServiceOperatorSecretName),
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret2",
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}
		secrets := &corev1.SecretList{Items: expectedSecrets}
		ns.Items = append(ns.Items, additionalNamespaces...)

		k8sClient := fake.NewClientBuilder().
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretProvider := NewSecretProvider(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretProvider.All(context.TODO())
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should fetch module's secret only", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		additionalNamespaces := createAdditionalNamespaces()
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      btpServiceOperatorSecretName,
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, additionalSecrets...)
		ns.Items = append(ns.Items, additionalNamespaces...)

		k8sClient := fake.NewClientBuilder().
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretProvider := NewSecretProvider(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretProvider.All(context.TODO())
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should fetch namespace prefixed secret only", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		additionalNamespaces := createAdditionalNamespaces()
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test2-%s", btpServiceOperatorSecretName),
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, additionalSecrets...)
		ns.Items = append(ns.Items, additionalNamespaces...)

		k8sClient := fake.NewClientBuilder().
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretProvider := NewSecretProvider(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretProvider.All(context.TODO())
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should fetch only secrets referenced in service instances", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		additionalNamespaces := createAdditionalNamespaces()
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret2",
					Namespace: controllers.ChartNamespace,
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, additionalSecrets...)
		ns.Items = append(ns.Items, additionalNamespaces...)

		k8sClient := fake.NewClientBuilder().
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretProvider := NewSecretProvider(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretProvider.All(context.TODO())
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should return nil when there are no btp operator secrets", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		additionalNamespaces := createAdditionalNamespaces()
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: additionalSecrets}
		ns.Items = append(ns.Items, additionalNamespaces...)

		k8sClient := fake.NewClientBuilder().
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretProvider := NewSecretProvider(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretProvider.All(context.TODO())
		require.NoError(t, err)

		// then
		assert.Nil(t, actualSecrets)
	})
}

func createAdditionalNamespaces() []corev1.Namespace {
	return []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: controllers.ChartNamespace,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test2",
			},
		},
	}
}

func createAdditionalSecrets() []corev1.Secret {
	return []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unrelated-additional-secret1",
				Namespace: controllers.ChartNamespace,
			},
			StringData: map[string]string{
				"foo": "bar",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unrelated-additional-secret2",
				Namespace: "test1",
			},
			StringData: map[string]string{
				"foo": "bar",
			},
		},
	}
}

func secretNameIndexer(obj client.Object) []string {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		panic(fmt.Errorf("indexer function for type %T's metadata.name field received"+
			" object of type %T, this should never happen", corev1.Secret{}, obj))
	}

	return []string{secret.Name}
}

func compareSecretSlices(t *testing.T, expected, actual []corev1.Secret) {
	assert.Equal(t, len(expected), len(actual))

	for _, expectedSecret := range expected {
		if !containsSecret(actual, expectedSecret) {
			t.Errorf("Expected secret %s not found in the actual list", expectedSecret.Name)
		}
	}
}

func containsSecret(secrets []corev1.Secret, secret corev1.Secret) bool {
	for _, s := range secrets {
		if s.Name == secret.Name && s.Namespace == secret.Namespace {
			return true
		}
	}
	return false
}
