# Informer's Cache

The controller manager uses informers with a cache. All observed resources (btpoperator, Secret, ConfigMap etc.) are stored in the cache. Because of the out of memory risk, the cache is configured with a label selector `app.kubernetes.io/managed-by in (btp-manager,kcp-kyma-environment-broker)`. See [cache.go](../../controllers/cache.go).