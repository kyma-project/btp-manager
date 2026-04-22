# SAP BTP Operator Busola Extension

## Overview

The SAP BTP Operator Busola extension adds a dedicated UI view for the `BtpOperator` custom resource (CR) in [Busola](https://github.com/kyma-project/busola), the Kyma web console. The extension is defined as a Kubernetes ConfigMap and uses the [Busola extensibility](https://github.com/kyma-project/busola/tree/main/docs/contributor/extensibility) mechanism.

The extension ConfigMap is located at [`config/busola-extension/sap-btp-operator-extension.yaml`](../../config/busola-extension/sap-btp-operator-extension.yaml).

## UI Structure

The extension adds a detail view for the `BtpOperator` CR under **Kyma > BTP Operators** in the namespace navigation. The view consists of the following sections: Header, BTP Operator Secrets Panel, Namespace-Based Secrets Panel, and Custom Secrets Panel.

### Header

The header (Metadata card) shows the following:

| Field | Description |
|---|---|
| **Documentation** | External link to the [SAP BTP Operator Module](https://help.sap.com/docs/btp/sap-business-technology-platform/sap-btp-operator-module) documentation |
| **BTP Manager Version** | Image tag of the `btp-manager-controller-manager` deployment |
| **SAP BTP Service Operator Version** | **APP_VERSION** environment variable of the `sap-btp-operator-controller-manager` deployment |
| **Custom Resource Definition** | Link to the `btpoperators.operator.kyma-project.io` CRD |
| **Service Bindings** | Count of all ServiceBinding resources in the cluster, linking to the `servicebindings.services.cloud.sap.com` CRD |
| **Service Instances** | Count of all ServiceInstance resources in the cluster, linking to the `serviceinstances.services.cloud.sap.com` CRD |

### SAP BTP Operator Secrets Panel

The **SAP BTP Operator Secrets** panel aggregates information about Secrets used by the SAP BTP Operator module. It contains two sub-panels.

**BTP Manager Secret**

Shows details of the `sap-btp-manager` Secret in the `kyma-system` namespace:

| Field | Description |
|---|---|
| **Status** | `Managed` if the secret is controlled by Kyma; `Unmanaged` if `kyma-project.io/skip-reconciliation: true` is set |
| **Credentials Namespace** | Namespace where the SAP BTP service operator looks for Secrets; decoded from `data.credentials_namespace`, defaults to `kyma-system` |
| **Cluster ID** | Decoded `data.cluster_id`, links to the SAP BTP cockpit |
| **Service Manager URL** | Decoded `data.sm_url` |
| **Token URL** | Decoded `data.tokenurl` |

The **Edit** link navigates directly to the `sap-btp-manager` Secret in Busola.

**SAP BTP Service Operator Secret**

Shows the cluster-wide `sap-btp-service-operator` Secret (the default Secret injected from the BTP Manager Secret). Its status is always `Inherited`.

### Namespace-Based Secrets Panel

Lists all Secrets whose names match the pattern `*-sap-btp-service-operator` across all namespaces. These Secrets provide per-namespace credentials overrides.

| Column | Description |
|---|---|
| Name | Link to the secret in Busola |
| Namespace | Namespace where the secret exists |
| Status | `In Use` if the secret is in the credentials namespace; `Not in Use` otherwise |

### Custom Secrets Panel

Lists all ServiceInstance resources that reference a custom secret through `spec.btpAccessCredentialsSecret`. Secrets are grouped by name.

| Column | Description |
|---|---|
| Name | Link to the Secret in Busola |
| Namespace | Namespace of the first referencing ServiceInstance |
| Service Instances | Count of ServiceInstances referencing the Secret |
| Status | `In Use` if the Secret namespace matches the credentials namespace; `Not in Use` otherwise |

## Data Sources

The extension defines the following data sources in the **dataSources** field of the ConfigMap:

| Name | Resource | Purpose |
|---|---|---|
| `btpSecret` / `btpSecret2` / `btpSecret3` / `btpSecret4` | `sap-btp-manager` Secret | Multiple copies to ensure stable rendering across different panel scopes |
| `btpManagerDeployment` | `btp-manager-controller-manager` Deployment | BTP Manager version |
| `btpOperatorDeployment` | `sap-btp-operator-controller-manager` Deployment | SAP BTP service operator version |
| `defaultSecret` | All Secrets named `sap-btp-service-operator` (cluster-wide) | **SAP BTP Service Operator Secret** sub-panel |
| `namespacedSecrets` | All Secrets matching `*-sap-btp-service-operator` (cluster-wide) | **Namespace-Based Secrets** panel |
| `referencedSecrets` | All ServiceInstances with `spec.btpAccessCredentialsSecret` set | **Custom Secrets** panel |
| `allServiceInstances` | All ServiceInstances (cluster-wide) | Service instances count |
| `allSecrets` | All Secrets (cluster-wide) | Creation timestamps in **Custom Secrets** panel |

> [!NOTE]
> Multiple copies of the `sap-btp-manager` data source (`btpSecret`, `btpSecret2`, etc.) are intentional. Busola caches data source results per rendering scope, and using the same name across different panel contexts can cause stale data to appear.

## Modify the Extension

To apply a change to a running local test cluster, run:

```shell
kubectl apply -f config/busola-extension/sap-btp-operator-extension.yaml -n kube-public
```

Refresh the Busola browser tab to pick up the updated ConfigMap.

## Test the Extension

The Busola extension has a dedicated Cypress E2E test suite. See [`busola-tests/README.md`](../../busola-tests/README.md) for setup instructions and test scenario descriptions.

Tests run automatically in CI on pull requests that modify `config/busola-extension/**` or `busola-tests/**`. See [`.github/workflows/btp-operator-e2e.yaml`](../../.github/workflows/btp-operator-e2e.yaml) for the full workflow definition.
