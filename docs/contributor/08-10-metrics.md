# BTP Manager Metrics

## Overview
BTP Manager provides metrics on the endpoint `:8080/metrics`. You find Kubebuilder, Golang, and custom metrics there. They are collected by Prometheus.

## Custom Metrics Emitted by BTP Manager

| Metric                                   | Description                                                                                       |
|:-----------------------------------------|:--------------------------------------------------------------------------------------------------|
| **btpmanager_certs_regenerations_total**   | The total number of [certificate](06-10-certs.md) regenerations.                                                                            |
| **btpmanager_custom_config_applied**       | Gauge indicating if the custom configuration ConfigMap is applied (1 = applied, 0 = not applied).                                           |
| **btpmanager_credential_probe_status**     | Gauge indicating the [CA bundle probe](09-10-ca-bundle-probe.md) status: 1 = alert (CA mounted but token URL cert not trusted), 0 = ok.     |
