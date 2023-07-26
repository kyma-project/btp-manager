# Informer's cache

The controller manager uses informers with cache. All observed resources (btpoperator, secret, configmap etc.) are stored in the cache. Because of out of memory risk, the cache is configured with a label selector `app.kubernetes.io/managed-by in (btp-manager,kcp-kyma-environment-broker)`. See [cache.go](controllers/cache.go).