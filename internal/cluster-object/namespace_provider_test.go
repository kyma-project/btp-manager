package clusterobject

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNamespaceProvider(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("should fetch all namespaces", func(t *testing.T) {
		// given
		namespaces := initNamespaces()
		k8sClient := fake.NewClientBuilder().WithLists(namespaces).Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)

		// when
		nsList, err := nsProvider.GetAll(context.TODO())

		// then
		if err != nil {
			t.Errorf("Error while fetching namespaces: %s", err)
		}
		assert.Len(t, nsList.Items, 4)
	})

	t.Run("should return error when no namespaces found", func(t *testing.T) {
		// given
		k8sClient := fake.NewClientBuilder().Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)

		// when
		_, err := nsProvider.GetAll(context.TODO())

		// then
		require.Error(t, err)
	})

	t.Run("should fetch namespaces by labels", func(t *testing.T) {
		// given
		namespaces := initNamespaces()
		k8sClient := fake.NewClientBuilder().WithLists(namespaces).Build()
		nsProvider := NewNamespaceProvider(k8sClient, logger)

		// when
		nsList, err := nsProvider.GetAllByLabels(context.TODO(), map[string]string{"test": "test"})

		// then
		if err != nil {
			t.Errorf("Error while fetching namespaces by labels: %s", err)
		}
		assert.Len(t, nsList.Items, 1)
	})
}

func initNamespaces() *corev1.NamespaceList {
	return &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kyma-system",
					Labels: map[string]string{
						"test": "test",
					},
				},
			},
		},
	}
}
