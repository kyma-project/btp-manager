package clusterobject

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const logComponentNameKey = "component"

type Provider[L client.ObjectList, O client.Object] interface {
	ClusterScopedProvider[L]
	NamespacedProvider[O]
}

type ClusterScopedProvider[T client.ObjectList] interface {
	GetAll(ctx context.Context) (T, error)
	GetAllByLabels(ctx context.Context, labels map[string]string) (T, error)
}

type NamespacedProvider[T client.Object] interface {
	GetByNameAndNamespace(ctx context.Context, name, namespace string) (T, error)
}
