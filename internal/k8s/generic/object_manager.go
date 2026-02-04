package generic

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	logger := log.FromContext(ctx)
	logger.Info("Creating object", "name", object.GetName())
	if err := m.client.Create(ctx, object, opts...); err != nil {
		return fmt.Errorf("while creating %q: %w", object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T]) Apply(ctx context.Context, object T, opts ...client.PatchOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Applying object", "name", object.GetName())
	if err := m.client.Patch(ctx, object, client.Apply, opts...); err != nil {
		return fmt.Errorf("while applying %q: %w", object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T]) Get(ctx context.Context, key client.ObjectKey, object T, opts ...client.GetOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Getting object", "name", key.Name)
	if err := m.client.Get(ctx, key, object, opts...); err != nil {
		return fmt.Errorf("while getting %q: %w", key.Name, err)
	}
	return nil
}

func (m *ObjectManager[T]) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Listing objects")
	if err := m.client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("while listing objects: %w", err)
	}
	return nil
}

func (m *ObjectManager[T]) Update(ctx context.Context, object T, opts ...client.UpdateOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating object", "name", object.GetName())
	if err := m.client.Update(ctx, object, opts...); err != nil {
		return fmt.Errorf("while updating %q: %w", object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T]) Delete(ctx context.Context, object T, opts ...client.DeleteOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Deleting object", "name", object.GetName())
	if err := m.client.Delete(ctx, object, opts...); err != nil {
		return fmt.Errorf("while deleting %q: %w", object.GetName(), err)
	}
	return nil
}
