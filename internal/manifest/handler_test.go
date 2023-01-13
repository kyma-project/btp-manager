package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

const resourcesDir = "testdata"

func TestHandler_CollectObjectsFromDir(t *testing.T) {
	// given
	scheme := clientgoscheme.Scheme
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	handler := Handler{Scheme: scheme}
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
	assert.Equal(t, 3, len(objs), "Should match number of manifests in test_manifests.yml")
	assert.Contains(t, objs, configMapAsRuntimeObject)
}
