# Service Binding Rotation

Enhance security by automatically rotating the credentials associated with your service bindings. This process involves generating a new service binding while keeping the old credentials active for a specified period to ensure a smooth transition.

## Enable Automatic Rotation

To enable automatic service binding rotation, use the **credentialsRotationPolicy** field within the `spec` section of the ServiceBinding resource. The field allows you to configure the following parameters:

| Parameter         | Type     | Description                                                                                                                               | Valid Values |
|-----------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------|--------------|
| **enabled**           | bool    | Turns automatic rotation on or off.                                                                                                    | None                    |
| **rotationFrequency** | string  | Defines the desired interval between binding rotations.             | "m" (minute), "h" (hour)|
| **rotatedBindingTTL** | string  | Determines how long to keep the old ServiceBinding resource after rotation and before deletion. The actual TTL may be slightly longer. | "m" (minute), "h" (hour) |   

> [!NOTE] 
> The `credentialsRotationPolicy` does not manage the validity or expiration of the credentials themselves. This is determined by the service you are using.

## Rotation Process

The `credentialsRotationPolicy` is evaluated periodically during a [control loop](https://kubernetes.io/docs/concepts/architecture/controller/) on every service binding update or during a complete reconciliation process. This means the actual rotation occurs in the closest upcoming reconciliation loop. 

## Immediate Rotation

You can trigger an immediate rotation regardless of the configured **rotationFrequency** by adding the `services.cloud.sap.com/forceRotate: "true"` annotation to the ServiceBinding resource. The immediate rotation only works if automatic rotation is already enabled. 

The following example shows the configuration of a ServiceBinding resource for rotating credentials every 25 days (600 hours) and keeping the old ServiceBinding resource for 2 days (48 hours) before deleting it:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: sample-binding
spec:
  serviceInstanceName: sample-instance
  credentialsRotationPolicy:
    enabled: true
    rotatedBindingTTL: 48h
    rotationFrequency: 600h
 ```

## After Rotation

Once the ServiceBinding is rotated:
* The Secret is updated with the latest credentials. 
* The old credentials are kept in a newly-created Secret named `original-secret-name(variable)-guid(variable)`.
This temporary Secret is kept until the configured deletion time (TTL) expires.

## Check Last Rotation

To view the timestamp of the last service binding rotation, refer to the **status.lastCredentialsRotationTime** field.

## Limitations

Automatic credential rotation cannot be enabled for a backup ServiceBinding (named: original-binding-name(variable)-guid(variable)) marked with the `services.cloud.sap.com/stale` label.
This backup service binding is created during the credentials rotation process to facilitate the process.

# Passing Parameters <!--where should this topic go, if anywhere?-->
To set input parameters, go to the `spec` of the ServiceInstance or ServiceBinding resource, and use both or one of the following fields:
* **parameters**: Specifies a set of properties sent to the service broker.
  The specified data is passed to the service broker without any modifications - aside from converting it to JSON for transmission to the broker if the `spec` field is specified as YAML.
  All valid YAML or JSON constructs are supported. 
  > [!NOTE] 
  > Only one parameter field per `spec` can be specified.<!--can this be changed to active?-->
* **parametersFrom**: Specifies which Secret, together with the key in it, to include in the set of parameters sent to the broker.
  The key contains a `string` that represents the JSON. The **parametersFrom** field is a list that supports multiple sources referenced per `spec`.

If multiple sources in the **parameters** and **parametersFrom** fields are specified,
the final payload results from merging all of them at the top level.
If there are any duplicate properties defined at the top level, the specification
is considered to be invalid. The further processing of the ServiceInstance or ServiceBinding
resource stops with the status `Error`.

See the following example of the `spec` format in YAML:
```yaml
spec:
  ...
  parameters:
    name: value
  parametersFrom:
    - secretKeyRef:
        name: my-secret
        key: secret-parameter
```

See the following example of the `spec` format in JSON:
```json
{
  "spec": {
    "parameters": {
      "name": "value"
    },
    "parametersFrom": {
      "secretKeyRef": {
        "name": "my-secret",
        "key": "secret-parameter"
      }
    }
  } 
}
```
The Secret with the key named **secret-parameter**: <!--what's this one about?-->
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
type: Opaque
stringData:
  secret-parameter:
    '{
      "password": "password"
    }'
```
The final JSON payload to send to the broker:
```json
{
  "name": "value",
  "password": "password"
}
```

To list multiple parameters in the Secret, separate key-value pairs with commas. See the following example:
```yaml
secret-parameter:
  '{
    "password": "password",
    "key2": "value2",
    "key3": "value3"
  }'
```
