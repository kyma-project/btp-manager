package clusterobject

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider interface {
	All() (client.ObjectList, error)
	ByNameAndNamespace(name, namespace string) (client.Object, error)
}
