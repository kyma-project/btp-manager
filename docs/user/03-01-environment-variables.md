# Environment Variables

This document describes the environment variables used by the BTP Manager controller.

| Environment Variable | Required | Example | Usage |
|---------------------|----------|---------|-------|
| SAP_BTP_SERVICE_OPERATOR | Yes | `europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/sap/sap-btp-service-operator/controller:v0.9.3` | Specifies the container image for the SAP BTP Service Operator component. The controller reads this environment variable when deploying the SAP BTP Service Operator and sets it as the image for the `manager` container in the deployment. |
| KUBE_RBAC_PROXY | Yes | `europe-docker.pkg.dev/kyma-project/prod/external/quay.io/brancz/kube-rbac-proxy:v0.20.0` | Specifies the container image for the Kubernetes RBAC Proxy component. The controller reads this environment variable when deploying the SAP BTP Service Operator and sets it as the image for the `kube-rbac-proxy` container in the deployment. |
| SKR_IMG_PULL_SECRET | No | `my-registry-secret` | Specifies the name of an image pull secret to be used for pulling container images. When set, the controller configures the deployment to use this secret for authenticating with container registries when pulling images. This is particularly useful when using private container registries that require authentication. |

## Setting Environment Variables

### In Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: btp-manager-controller
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: SAP_BTP_SERVICE_OPERATOR
          value: "europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/sap/sap-btp-service-operator/controller:v0.9.3"
        - name: KUBE_RBAC_PROXY
          value: "europe-docker.pkg.dev/kyma-project/prod/external/quay.io/brancz/kube-rbac-proxy:v0.20.0"
        - name: SKR_IMG_PULL_SECRET
          value: "my-registry-secret"
```

### In Docker/Local Development

```bash
export SAP_BTP_SERVICE_OPERATOR="europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/sap/sap-btp-service-operator/controller:v0.9.3"
export KUBE_RBAC_PROXY="europe-docker.pkg.dev/kyma-project/prod/external/quay.io/brancz/kube-rbac-proxy:v0.20.0"
export SKR_IMG_PULL_SECRET="my-registry-secret"  # optional
```

## Image Management

The BTP Manager controller dynamically sets the container images for the SAP BTP Service Operator deployment based on the values of these environment variables. This allows for:

1. **Version Control**: Easy updating of component versions by changing environment variable values
2. **Registry Flexibility**: Using different container registries without code changes
3. **Security**: Supporting private registries through image pull secrets

When the controller reconciles a BtpOperator resource, it reads these environment variables and applies them to the appropriate containers in the SAP BTP Service Operator deployment manifest.
