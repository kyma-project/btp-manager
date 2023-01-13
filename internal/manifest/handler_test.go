package manifest

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	resourcesDir           = "testdata"
	manifestsYamlFile      = "test-manifests.yaml"
	singleManifestYamlFile = "test-secret.yml"
)

func TestHandler_SuccessPath(t *testing.T) {
	// suite setup
	scheme := clientgoscheme.Scheme
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	handler := Handler{Scheme: scheme}

	t.Run("should get manifests from single yaml file", func(t *testing.T) {
		// given
		yamlPath := fmt.Sprintf("%s%c%s", resourcesDir, os.PathSeparator, manifestsYamlFile)

		// when
		manifests, err := handler.GetManifestsFromYaml(yamlPath)
		require.NoError(t, err)

		// then
		assert.NotEmpty(t, manifests)
		assert.Len(t, manifests, 3, "should match number of manifests in single yaml file")
	})

	t.Run("should create object from manifest", func(t *testing.T) {
		// given
		expectedGvk := schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
		expectedKeyName, expectedKeyValue := "clientid", "test_clientid"
		yamlPath := fmt.Sprintf("%s%c%s", resourcesDir, os.PathSeparator, singleManifestYamlFile)
		data, err := os.ReadFile(yamlPath)
		require.NoError(t, err)

		// when
		obj, err := handler.CreateObjectFromManifest(string(data))
		require.NoError(t, err)

		// then
		assert.Equal(t, obj.GetObjectKind().GroupVersionKind(), expectedGvk)

		// when
		secret := obj.(*corev1.Secret)

		// then
		assert.NotEmpty(t, secret.Data)
		assert.Equal(t, string(secret.Data[expectedKeyName]), expectedKeyValue)
	})

	t.Run("should create objects from yamls", func(t *testing.T) {
		// given

		var configMapAsRuntimeObject runtime.Object = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "default",
			},
			Data: map[string]string{"text": "test"},
		}

		// when
		objs, err := handler.CollectObjectsFromDir(resourcesDir)
		require.NoError(t, err)

		// then
		assert.NotEmpty(t, objs)
		assert.Len(t, objs, 4, "should match number of manifests in all yaml files")
		assert.Contains(t, objs, configMapAsRuntimeObject)
	})
}
