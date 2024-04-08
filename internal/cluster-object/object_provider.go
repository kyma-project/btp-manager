package clusterobject

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const logComponentNameKey = "component"

type Provider[T client.Object | client.ObjectList] interface {
	ClusterScopedProvider[T]
	NamespacedProvider[T]
}

type ClusterScopedProvider[T client.ObjectList] interface {
	All() (T, error)
}

type NamespacedProvider[T client.Object] interface {
	GetByNameAndNamespace(name, namespace string) (T, error)
}
