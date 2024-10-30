package clusterobject

import (
	"context"
	"errors"
	"fmt"
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

func (p *NamespaceProvider) GetAll(ctx context.Context) (*v1.NamespaceList, error) {
	p.logger.Info("fetching all namespaces")

	namespaces := &v1.NamespaceList{}
	if err := p.Reader.List(ctx, namespaces); err != nil {
		p.logger.Error("failed to fetch all namespaces", "error", err)
		return nil, err
	}

	if len(namespaces.Items) == 0 {
		err := errors.New("no namespaces found")
		p.logger.Error(err.Error())
	}

	return namespaces, nil
}

func (p *NamespaceProvider) GetAllByLabels(ctx context.Context, labels map[string]string) (*v1.NamespaceList, error) {
	p.logger.Info("fetching namespaces by labels")
	namespaces := &v1.NamespaceList{}
	err := p.List(ctx, namespaces, client.MatchingLabels(labels))
	if err != nil {
		p.logger.Error("while fetching namespaces by labels", "error", err, "labels", labels)
		return nil, err
	}

	if len(namespaces.Items) == 0 {
		p.logger.Warn(fmt.Sprintf("no namespaces found with labels: %v", labels))
		return nil, err
	}

	return namespaces, err
}
