# Network Policies

The SAP BTP Operator module can create network policies to control traffic for the SAP BTP service operator Pods. By default, network policies are disabled.

## Enable Network Policies

To enable network policies for SAP BTP Operator, run:

```bash
kubectl patch btpoperators/btpoperator -n kyma-system --type='merge' -p='{"spec":{"networkPoliciesEnabled":true}}'
```

## Disable Network Policies

To disable network policies, run:

```bash
kubectl patch btpoperators/btpoperator -n kyma-system --type='merge' -p='{"spec":{"networkPoliciesEnabled":false}}'
```

## What Each Policy Does

When enabled, the following network policies are created for the SAP BTP Operator module:

| Policy Name | Description |
|-------------|----------------|
| `kyma-project.io--btp-operator-allow-to-apiserver` | Egress from BTP operator pods to any destination on TCP port 443 (e.g., Kubernetes API server) |
| `kyma-project.io--btp-operator-to-dns` | Egress from BTP operator Pods to DNS services (UDP/TCP port 53, 8053) for cluster and external DNS resolution |
| `kyma-project.io--allow-btp-operator-metrics` | Ingress to BTP operator Pods on TCP port 8080 from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` (metrics scraping) |
| `kyma-project.io--btp-operator-allow-to-webhook` | Ingress to BTP operator Pods on TCP port 9443 (webhook server) from any source |

## Verify Status

To check if network policies are active, run:

```bash
kubectl get networkpolicies -n kyma-system -l kyma-project.io/managed-by=btp-manager
```
