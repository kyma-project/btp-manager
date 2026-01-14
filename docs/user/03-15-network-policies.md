# Network Policies

Learn about the network policies for the SAP BTP Operator module and how to manage them.

## Default Network Policies

By default, the SAP BTP Operator module creates the following network policies to control traffic to and from the SAP BTP service operator Pods. <!--should we mention anything about the security-related purpose of the network policies?-->

- `kyma-project.io--btp-operator-allow-to-apiserver`: Allows egress from the SAP BTP Operator module Pods to any destination on TCP port 443 (for example, Kubernetes API server)
- `kyma-project.io--btp-operator-to-dns`: Allows egress from the SAP BTP Operator module Pods to DNS services (UDP/TCP port 53, 8053) for cluster and external DNS resolution
- `kyma-project.io--allow-btp-operator-metrics`: Allows ingress to the SAP BTP Operator module Pods on TCP port 8080 from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` (metrics scraping)
- `kyma-project.io--btp-operator-allow-to-webhook`: Allows ingress to the SAP BTP Operator module Pods on TCP port 9443 (webhook server) from any source

## Disable Network Policies

To disable network policies for SAP BTP Operator, add the following annotation to the BtpOperator custom resource (CR):

```bash
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies=true
```
<!--Should a warning be added here to inform about the consequences of disabling the policies?-->
## Enable Network Policies

To enable network policies, remove the following annotation:

```bash
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies-
```

## Verify Status

To check if the network policies are active, run:

```bash
kubectl get networkpolicies -n kyma-system -l kyma-project.io/managed-by=btp-manager
```
