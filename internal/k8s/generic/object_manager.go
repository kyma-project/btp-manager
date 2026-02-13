package generic

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ObjectManager[T client.Object, U client.ObjectList] struct {
	client client.Client
}

func NewObjectManager[T client.Object, U client.ObjectList](k8sClient client.Client) *ObjectManager[T, U] {
	return &ObjectManager[T, U]{
		client: k8sClient,
	}
}

func (m *ObjectManager[T, U]) Create(ctx context.Context, object T, opts ...client.CreateOption) error {
	logger := log.FromContext(ctx)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	logger.Info("Creating object", "kind", kind, "name", object.GetName())
	if err := m.client.Create(ctx, object, opts...); err != nil {
		return fmt.Errorf("while creating %s %q: %w", kind, object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T, U]) Apply(ctx context.Context, object T, opts ...client.PatchOption) error {
	logger := log.FromContext(ctx)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	logger.Info("Applying object", "kind", kind, "name", object.GetName())
	if err := m.client.Patch(ctx, object, client.Apply, opts...); err != nil {
		return fmt.Errorf("while applying %s %q: %w", kind, object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T, U]) Get(ctx context.Context, key client.ObjectKey, object T, opts ...client.GetOption) error {
	logger := log.FromContext(ctx)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	logger.Info("Getting object", "kind", kind, "name", key.Name)
	if err := m.client.Get(ctx, key, object, opts...); err != nil {
		return fmt.Errorf("while getting %s %q: %w", kind, key.Name, err)
	}
	return nil
}

func (m *ObjectManager[T, U]) List(ctx context.Context, list U, opts ...client.ListOption) error {
	logger := log.FromContext(ctx)
	logger.Info("Listing objects", "kind", list.GetObjectKind().GroupVersionKind().Kind)
	if err := m.client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("while listing objects: %w", err)
	}
	return nil
}

func (m *ObjectManager[T, U]) Update(ctx context.Context, object T, opts ...client.UpdateOption) error {
	logger := log.FromContext(ctx)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	logger.Info("Updating object", "kind", kind, "name", object.GetName())
	if err := m.client.Update(ctx, object, opts...); err != nil {
		return fmt.Errorf("while updating %s %q: %w", kind, object.GetName(), err)
	}
	return nil
}

func (m *ObjectManager[T, U]) Delete(ctx context.Context, object T, opts ...client.DeleteOption) error {
	logger := log.FromContext(ctx)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	logger.Info("Deleting object", "kind", kind, "name", object.GetName())
	if err := m.client.Delete(ctx, object, opts...); err != nil {
		return fmt.Errorf("while deleting %s %q: %w", kind, object.GetName(), err)
	}
	return nil
}
