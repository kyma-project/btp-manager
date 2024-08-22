package clusterobject

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager[L client.ObjectList, O client.Object] interface {
	Provider[L, O]
	Creator[O]
	Deleter[L, O]
}

type Creator[T client.Object] interface {
	Create(ctx context.Context, obj T) error
}

type Deleter[L client.ObjectList, O client.Object] interface {
	Delete(ctx context.Context, obj O) error
	DeleteList(ctx context.Context, objs L) error
	DeleteAllByLabels(ctx context.Context, labels map[string]string) error
}
