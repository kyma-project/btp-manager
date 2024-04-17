package clusterobject

import (
	"context"
	"errors"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const namespaceProviderName = "NamespaceProvider"

type NamespaceProvider struct {
	client.Reader
	logger *slog.Logger
}

func NewNamespaceProvider(reader client.Reader, logger *slog.Logger) *NamespaceProvider {
	logger = logger.With(logComponentNameKey, namespaceProviderName)

	return &NamespaceProvider{
		Reader: reader,
		logger: logger,
	}
}

func (p *NamespaceProvider) All(ctx context.Context) (*v1.NamespaceList, error) {
	p.logger.Info("fetching all namespaces")

	namespaces := &v1.NamespaceList{}
	if err := p.Reader.List(ctx, namespaces); err != nil {
		p.logger.Error("failed to fetch all namespaces", "error", err)
		return nil, err
	}

	if len(namespaces.Items) == 0 {
		err := errors.New("no namespaces found")
		p.logger.Error(err.Error())
		return nil, err
	}

	return namespaces, nil
}
