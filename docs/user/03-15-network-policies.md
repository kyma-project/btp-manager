# Network Policies

The SAP BTP Operator module can create network policies to control traffic for the SAP BTP service operator Pods. By default, network policies are enabled.

## Disable Network Policies

To disable network policies for SAP BTP Operator, add the following annotation to the BtpOperator custom resource:

```bash
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies=true
```

## Enable Network Policies

To enable network policies remove the annotation:

```bash
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies-
```

## What Each Policy Does

By default, the following network policies are created for the SAP BTP Operator module:

| Policy Name | Description |
|-------------|----------------|
| `kyma-project.io--btp-operator-allow-to-apiserver` | Egress from the SAP BTP Operator module Pods to any destination on TCP port 443 (for example, Kubernetes API server) |
| `kyma-project.io--btp-operator-to-dns` | Egress from the SAP BTP Operator module Pods to DNS services (UDP/TCP port 53, 8053) for cluster and external DNS resolution |
| `kyma-project.io--allow-btp-operator-metrics` | Ingress to the SAP BTP Operator module Pods on TCP port 8080 from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` (metrics scraping) |
| `kyma-project.io--btp-operator-allow-to-webhook` | Ingress to the SAP BTP Operator module Pods on TCP port 9443 (webhook server) from any source |

## Verify Status

To check if network policies are active, run:

```bash
kubectl get networkpolicies -n kyma-system -l kyma-project.io/managed-by=btp-manager
```
