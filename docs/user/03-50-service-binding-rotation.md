# Service Binding Rotation

Enhance security by automatically rotating the credentials associated with your service bindings. This process involves generating a new service binding while keeping the old credentials active for a specified period to ensure a smooth transition.

## Enable Automatic Rotation

To enable automatic service binding rotation, use the **credentialsRotationPolicy** field within the `spec` section of the ServiceBinding resource. The field allows you to configure several parameters:

| Parameter         | Type     | Description                                                                                                                               | Valid Values |
|-----------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------|--------------|
| **enabled**           | bool    | Turns automatic rotation on or off.                                                                                                    | None                    |
| **rotationFrequency** | string  | Defines the desired interval between binding rotations.             | "m" (minute), "h" (hour)|
| **rotatedBindingTTL** | string  | Determines how long to keep the old ServiceBinding resource after rotation and before deletion. The actual TTL may be slightly longer. | "m" (minute), "h" (hour) |   

> [!NOTE] 
> The `credentialsRotationPolicy` does not manage the validity or expiration of the credentials themselves. This is determined by the specific service you are using.

## Rotation Process

The `credentialsRotationPolicy` is evaluated periodically during a [control loop](https://kubernetes.io/docs/concepts/architecture/controller/), which runs on every service binding update or during a full reconciliation process. This means the actual rotation occurs in the closest upcoming reconciliation loop. 

## Immediate Rotation

You can trigger an immediate rotation regardless of the configured **rotationFrequency** by adding the `services.cloud.sap.com/forceRotate: "true"` annotation to the ServiceBinding resource. The immediate rotation only works if automatic rotation is already enabled. 

The following example shows the configuration of a ServiceBinding resource to rotate credentials every 25 days (600 hours) and keep the old ServiceBinding resource for 2 days (48 hours) before deleting it:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: {BINDING_NAME}
spec:
  serviceInstanceName: {INSTANCE_NAME}
  credentialsRotationPolicy:
    enabled: true
    rotatedBindingTTL: 48h
    rotationFrequency: 600h
 ```

## After Rotation

Once the ServiceBinding is rotated:
* The Secret is updated with the latest credentials. 
* The old credentials are kept in a newly-created Secret named `original-secret-name(variable)-guid(variable)`. <!-- HOW CAN I CHANGE IT using a placeholder?-->
This temporary Secret is kept until the configured deletion time (TTL) expires.

## Check Last Rotation

To view the timestamp of the last service binding rotation, refer to the **status.lastCredentialsRotationTime** field.

## Limitations

Automatic credential rotation cannot be enabled for a backup ServiceBinding (named: original-binding-name(variable)-guid(variable <!-- HOW CAN I CHANGE IT using  placeholders?-->)) which is marked with the `services.cloud.sap.com/stale` label.
This backup service binding is created during the credentials rotation process to facilitate the process.

## Passing Parameters <!--READ THIS AGAIN!!!-->
To set input parameters, go to the `spec` field of the ServiceInstance or ServiceBinding resource, and use both or one of the following
fields:
- **parameters**: can be used to specify a set of properties to be sent to the
  broker. The data specified is passed to the service broker without any
  modifications - aside from converting it to JSON for transmission to the broker
  in the case of the `spec` field being specified as `YAML`. Any valid `YAML` or
  `JSON` constructs are supported. Only one parameter field may be specified per
  `spec`.
- **parametersFrom**: can be used to specify which Secret, and key in that Secret,
  which contains a `string` that represents the JSON to include in the set of
  parameters to be sent to the broker. The **parametersFrom** field is a list that
  supports multiple sources referenced per `spec`.

If multiple sources in the **parameters** and **parametersFrom** fields are specified,
the final payload is a result of merging all of them at the top level.
If there are any duplicate properties defined at the top level, the specification
is considered to be invalid. The further processing of the ServiceInstance or ServiceBinding
resource stops and its `status` is marked with an error condition.

The format of the `spec` in YAML
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

The format of the `spec` in JSON
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
The `secret` with the `secret-parameter`- named key:
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

You can list multiple parameters in the `secret`. To do so, separate "key": "value" pairs with commas as in this example:
```yaml
secret-parameter:
  '{
    "password": "password",
    "key2": "value2",
    "key3": "value3"
  }'
```
