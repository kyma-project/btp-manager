package clusterobject

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const logComponentNameKey = "component"

type ClusterScopedProvider[T client.ObjectList] interface {
	All(ctx context.Context) (T, error)
}

type NamespacedProvider[T client.Object] interface {
	GetByNameAndNamespace(ctx context.Context, name, namespace string) (T, error)
}
