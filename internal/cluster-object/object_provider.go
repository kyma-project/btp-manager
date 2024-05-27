package clusterobject

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const logComponentNameKey = "component"

type ClusterScopedProvider[T client.ObjectList] interface {
	All(ctx context.Context) (T, error)
}

type NamespacedProvider[T client.Object, TL client.ObjectList] interface {
	GetByNameAndNamespace(ctx context.Context, name, namespace string) (T, error)
	All(ctx context.Context) (TL, error)
}
