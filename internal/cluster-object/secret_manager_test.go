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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretManager(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheme := clientgoscheme.Scheme
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: siCrdName,
		},
	}

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
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAll(context.TODO())
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
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAll(context.TODO())
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
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAll(context.TODO())
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
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAll(context.TODO())
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
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAll(context.TODO())
		require.NoError(t, err)

		// then
		assert.Nil(t, actualSecrets)
	})

	t.Run("should fetch secrets by labels", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: controllers.ChartNamespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret2",
					Namespace: controllers.ChartNamespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, additionalSecrets...)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		actualSecrets, err := secretManager.GetAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should create a secret", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		secretToCreate := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1",
				Namespace: controllers.ChartNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			StringData: map[string]string{
				"foo": "bar",
			},
		}
		expectedSecrets := []corev1.Secret{secretToCreate}
		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: additionalSecrets}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		err := secretManager.Create(context.TODO(), &secretToCreate)
		require.NoError(t, err)

		// when
		actualSecrets, err := secretManager.GetAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should delete a secret", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		secretToDelete := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret2",
				Namespace: controllers.ChartNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			StringData: map[string]string{
				"foo": "bar",
			},
		}
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: controllers.ChartNamespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}

		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, secretToDelete)
		secrets.Items = append(secrets.Items, additionalSecrets...)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		err := secretManager.Delete(context.TODO(), &secretToDelete)
		require.NoError(t, err)

		// when
		actualSecrets, err := secretManager.GetAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should delete secrets from list", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		secretsToDelete := &corev1.SecretList{
			Items: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: controllers.ChartNamespace,
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					StringData: map[string]string{
						"foo": "bar",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret2",
						Namespace: controllers.ChartNamespace,
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					StringData: map[string]string{
						"foo": "bar",
					},
				},
			},
		}
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret3",
					Namespace: controllers.ChartNamespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}

		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, secretsToDelete.Items...)
		secrets.Items = append(secrets.Items, additionalSecrets...)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		err := secretManager.DeleteList(context.TODO(), secretsToDelete)
		require.NoError(t, err)

		// when
		actualSecrets, err := secretManager.GetAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
	})

	t.Run("should delete secrets by labels", func(t *testing.T) {
		// given
		ns := initNamespaces()
		sis := initServiceInstances(t)
		secretsToDelete := &corev1.SecretList{
			Items: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: controllers.ChartNamespace,
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					StringData: map[string]string{
						"foo": "bar",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret2",
						Namespace: controllers.ChartNamespace,
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					StringData: map[string]string{
						"foo": "bar",
					},
				},
			},
		}
		expectedSecrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret3",
					Namespace: controllers.ChartNamespace,
					Labels: map[string]string{
						"keep": "me",
					},
				},
				StringData: map[string]string{
					"foo": "bar",
				},
			},
		}

		additionalSecrets := createAdditionalSecrets()
		secrets := &corev1.SecretList{Items: expectedSecrets}
		secrets.Items = append(secrets.Items, secretsToDelete.Items...)
		secrets.Items = append(secrets.Items, additionalSecrets...)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(crd).
			WithLists(ns, sis, secrets).
			WithIndex(&corev1.Secret{}, "metadata.name", secretNameIndexer).
			Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)
		siProvider := NewServiceInstanceProvider(k8sClient, logger)
		secretManager := NewSecretManager(k8sClient, nsProvider, siProvider, logger)

		// when
		err := secretManager.DeleteAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)

		// when
		actualSecrets, err := secretManager.GetAllByLabels(context.TODO(), map[string]string{"foo": "bar"})
		require.NoError(t, err)
		assert.Empty(t, actualSecrets.Items)

		// when
		actualSecrets, err = secretManager.GetAllByLabels(context.TODO(), map[string]string{"keep": "me"})
		require.NoError(t, err)

		// then
		compareSecretSlices(t, expectedSecrets, actualSecrets.Items)
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
