# Informer's Cache

The controller manager uses informers with a cache. All observed resources (BtpOperator, Secret, ConfigMap, etc.) are stored in the cache. Because of the out of memory risk, the cache is configured with the label selector `app.kubernetes.io/managed-by in (btp-manager,kcp-kyma-environment-broker)`. For details, see [`cache.go`](../../controllers/cache.go).