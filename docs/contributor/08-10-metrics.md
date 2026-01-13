# BTP Manager Metrics

## Overview
BTP Manager provides metrics on the endpoint `:8080/metrics`. You find Kubebuilder, Golang, and custom metrics there. They are collected by Prometheus.

## Custom Metrics Emitted by BTP Manager

| Metric                                          | Description                                                                      |
| :----------------------------------------------- | :------------------------------------------------------------------------------- |
| **btpmanager_certs_regenerations_total**        | The total number of [certificate](06-10-certs.md) regenerations.                 |
