# Pass Parameters

You can set input parameters for your resources.

## Procedure

To set input parameters, go to the `spec` of the ServiceInstance or ServiceBinding resource, and use both or one of the following fields:

* **parameters**: Specifies a set of properties sent to the service broker.
  The specified data is passed to the service broker without any modifications - aside from converting it to the JSON format for transmission to the broker if the `spec` field is specified as a YAML file.
  All valid YAML or JSON constructs are supported.

  > [!NOTE] 
  > Only one **parameter** field per `spec` can be specified.

* **parametersFrom**: Specifies which Secret, together with the key in it, to include in the set of parameters sent to the service broker.
  The key contains a `string` that represents a JSON file. The **parametersFrom** field is a list that supports multiple sources referenced per `spec`.
  The ServiceInstance resource can specify multiple related Secrets.

* **watchParametersFromChanges**: If set to `true`, any changes to the Secret values listed in **parametersFrom** trigger an automatic update of the ServiceInstance resource.
  By default, the field is set to `false`  and must not be used if **parametersFrom** is empty.

If you specified multiple sources in the **parameters** and **parametersFrom** fields, the final payload merges all of them at the top level.

If there are any duplicate properties defined at the top level, the specification is considered to be invalid. 
The further processing of the ServiceInstance or ServiceBinding resource stops with the status `Error`.

## Examples

See the following examples:

*  The `spec` format in YAML:

    ```yaml
    spec:
      ...
      parameters:
        name: value
      parametersFrom:
        - secretKeyRef:
            name: {SECRET_NAME}
            key: secret-parameter
    ```

* The `spec` format in JSON:

  ```json
  {
    "spec": {
      "parameters": {
        "name": "value"
      },
      "parametersFrom": {
        "secretKeyRef": {
          "name": "{SECRET_NAME}",
          "key": "secret-parameter"
        }
      }
    } 
  }
  ```

* A Secret with the key named **secret-parameter**:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {SECRET_NAME}
  type: Opaque
  stringData:
    secret-parameter:
      '{
        "password": "password"
      }'
  ```

* The final JSON payload sent to the service broker:

  ```json
  {
    "name": "value",
    "password": "password"
  }
  ```

* Multiple parameters in the Secret with key-value pairs separated with commas:

  ```yaml
  secret-parameter:
    '{
      "password": "password",
      "key2": "value2",
      "key3": "value3"
    }'
  ```
