# Pass Parameters

You can set input parameters for your resources.

To pass additional parameters stored outside your resource spec, create a Kubernetes Secret manually in the same namespace as your ServiceInstance or ServiceBinding, and reference it using the `parametersFrom` field.

## Procedure

To set input parameters, go to the `spec` of the ServiceInstance or ServiceBinding resource, and use one or both of the following fields:

* **parameters**: Specifies a set of properties sent to the service broker.
  The specified data is passed to the service broker without any modifications - aside from converting it to JSON for transmission to the broker if the `spec` field is specified as YAML.
  All valid YAML or JSON constructs are supported.
* **parametersFrom**: Specifies which Secret (that you create manually in the same namespace) and which key in it to include in the set of parameters sent to the service broker.
  The key contains a `string` that represents a JSON file. The **parametersFrom** field is a list that supports multiple sources referenced per `spec`.
  The ServiceInstance resource can specify multiple related Secrets.

For ServiceInstance resources, you can also use the following parameter:

* **watchParametersFromChanges**: Use this field together with **parametersFrom**.
  This field is only relevant for ServiceInstance resources because you cannot update ServiceBinding resources. Set it to `true` to have the ServiceInstance automatically reconcile whenever the referenced Secret changes.
  By default, it is set to `false`.

> [!CAUTION]
> For change detection to work, the Secret referenced in **parametersFrom** must have the label `services.cloud.sap.com/managed-by-sap-btp-operator: "true"`. Without this label, the SAP BTP Service Operator does not watch the Secret, and changes to it are not detected — even if **watchParametersFromChanges** is set to `true`.

If you specify multiple sources in the **parameters** and **parametersFrom** fields, the final payload merges all of them at the top level.
To avoid errors, do not use the same top-level parameter name in multiple sources in the **parameters** and **parametersFrom** fields.
Otherwise, the specification is invalid, and further processing of the ServiceInstance or ServiceBinding resources stops with the status `Error`.

## Examples

The following example shows a ServiceInstance `spec` that uses both `parameters` and `parametersFrom`:

```yaml
spec:
  ...
  parameters:
    name: value
  parametersFrom:
    - secretKeyRef:
        name: {SECRET_NAME}
        key: secret-parameter
  watchParametersFromChanges: true      
```

The Secret referenced by `parametersFrom` must be created in the same namespace as the ServiceInstance:

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
  secret-parameter:
    '{
      "password": "password",
      "key2": "value2",
      "key3": "value3"
    }'
```

The values from `parameters` and `parametersFrom` are merged into a single JSON payload sent to the service broker:

```json
{
  "name": "value",
  "password": "password",
  "key2": "value2",
  "key3": "value3"
}
```

## Related Information

[ServiceInstance Not Updated When Watched Parameters Secret Is Modified](troubleshooting/05-03-watched-secret-not-detected.md)
