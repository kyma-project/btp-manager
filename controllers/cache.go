package controllers

import (
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CacheCreator(conf *rest.Config, opts cache.Options) (cache.Cache, error) {
	labelSelector, err := labels.Parse("app.kubernetes.io/managed-by in (btp-manager,kcp-kyma-environment-broker)")
	if err != nil {
		panic(fmt.Sprintf("unable to parse label selector: %s", err))
	}
	objSelector := cache.ByObject{
		Label: labelSelector,
	}

	opts.ByObject = map[client.Object]cache.ByObject{
		&corev1.Secret{}:    objSelector,
		&corev1.ConfigMap{}: objSelector,
		&admissionregistrationv1.ValidatingWebhookConfiguration{}: objSelector,
		&admissionregistrationv1.MutatingWebhookConfiguration{}:   objSelector,
	}

	return cache.New(conf, opts)
}
