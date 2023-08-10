# BTP Manager metrics

The BTP Manager Controller Manager provides metrics on the endpoint `:8080/metrics`. There you will find `Kubebuilder`, `Golang` and custom metrics. They are collected by `Prometheus`.

## Custom metrics emitted by BTP Manager Controller Manager:

| Metric                                          | Description                                                                      |
| ----------------------------------------------- | :------------------------------------------------------------------------------- |
| **btpmanager_certs_regenerations_total**        | The total number of [certificate](06-10-certs.md) regenerations                                    |