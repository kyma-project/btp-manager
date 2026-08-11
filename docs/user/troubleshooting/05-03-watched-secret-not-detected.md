# ServiceInstance Not Updated When Watched Parameters Secret Is Modified

## Symptom

You have a service instance configured with **parametersFrom** referencing a Secret and `watchParametersFromChanges: true`, but updating the Secret does not trigger an update of the service instance.

## Cause

The SAP BTP service operator uses limited cache mode, which is enabled by default in the SAP BTP Operator module for Kyma. In this mode, the operator only watches Secrets that carry the label `services.cloud.sap.com/managed-by-sap-btp-operator: "true"`. A Secret without this label is not tracked, so changes to it go undetected — no update is triggered and no error is reported.

## Solution

Add the label `services.cloud.sap.com/managed-by-sap-btp-operator: "true"` to the parameters Secret.

You can do this imperatively:

```bash
kubectl label secret {SECRET_NAME} -n {NAMESPACE} services.cloud.sap.com/managed-by-sap-btp-operator=true
```

Or include the label in your Secret manifest.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  namespace: {NAMESPACE}
  labels:
    services.cloud.sap.com/managed-by-sap-btp-operator: "true"
type: Opaque
stringData:
  secret-parameter: |
    {
      "key": "value"
    }
```

Once the label is present, the operator detects it immediately — no restart or waiting is required — and reconciles the service instance on the next Secret change.
