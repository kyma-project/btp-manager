package clusterobject

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/btp-manager/controllers"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceInstanceProvider(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("should fetch all service instances", func(t *testing.T) {
		// given
		givenSiList := initServiceInstances(t)
		k8sClient := fake.NewClientBuilder().WithLists(givenSiList).Build()
		siProvider := NewServiceInstanceProvider(k8sClient, logger)

		// when
		sis, err := siProvider.All(context.TODO())

		// then
		if err != nil {
			t.Errorf("Error while fetching service instances: %s", err)
		}
		if len(sis.Items) != 4 {
			t.Errorf("Expected 4 service instances, got %d", len(sis.Items))
		}
	})

	t.Run("should fetch service instances with secret reference", func(t *testing.T) {
		// given
		givenSiList := initServiceInstances(t)
		k8sClient := fake.NewClientBuilder().WithLists(givenSiList).Build()
		siProvider := NewServiceInstanceProvider(k8sClient, logger)

		// when
		sis, err := siProvider.AllWithSecretRef(context.TODO())

		// then
		if err != nil {
			t.Errorf("Error while fetching service instances: %s", err)
		}
		if len(sis.Items) != 2 {
			t.Errorf("Expected 2 service instances, got %d", len(sis.Items))
		}
		for _, si := range sis.Items {
			secretRef, _, err := unstructured.NestedString(si.Object, "spec", secretRefKey)
			if err != nil {
				t.Errorf("Error while fetching secret ref from service instance: %s", err)
			}
			if secretRef == "" {
				t.Error("Expected secret ref, got empty value")
			}
		}
	})
}

func initServiceInstances(t *testing.T) *unstructured.UnstructuredList {
	siList := &unstructured.UnstructuredList{}
	siList.SetGroupVersionKind(controllers.InstanceGvk)
	siList.Items = []unstructured.Unstructured{
		initServiceInstance(t, "si1", "namespace1"),
		initServiceInstance(t, "si2", "namespace2"),
		initServiceInstance(t, "si3", "namespace3", "secret1"),
		initServiceInstance(t, "si4", "namespace3", "secret2"),
	}

	return siList
}

func initServiceInstance(t *testing.T, name, namespace string, secretRef ...string) unstructured.Unstructured {
	si := unstructured.Unstructured{}
	si.SetGroupVersionKind(controllers.InstanceGvk)
	si.SetName(name)
	si.SetNamespace(namespace)
	if len(secretRef) > 0 {
		err := unstructured.SetNestedField(si.Object, secretRef[0], "spec", secretRefKey)
		if err != nil {
			t.Errorf("error while setting secret ref: %s", err)
		}
	}
	return si
}
