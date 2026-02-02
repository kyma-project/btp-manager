package generic

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectManager[T client.Object] struct {
	client client.Client
}

func NewObjectManager[T client.Object](k8sClient client.Client) *ObjectManager[T] {
	return &ObjectManager[T]{
		client: k8sClient,
	}
}

func (m *ObjectManager[T]) Create(ctx context.Context, object T, opts ...client.CreateOption) error {
	if err := m.client.Create(ctx, object, opts...); err != nil {
		return fmt.Errorf("while creating %s %s: %w", object.GetObjectKind().GroupVersionKind().Kind, object.GetName(), err)
	}
	return nil
}
