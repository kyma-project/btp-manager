# Service Binding Rotation

Enhance security by automatically rotating the credentials associated with your service bindings. This process involves generating a new service binding while keeping the old credentials active for a specified period to ensure a smooth transition.

## Enable Automatic Rotation

To enable automatic service binding rotation, use the **credentialsRotationPolicy** field within the `spec` section of the ServiceBinding resource. The field allows you to configure several parameters:

| Parameter         | Type     | Description                                                                                                                            | Valid Values |
|-----------------|---------|---------------------------------------------------------------------------------------------------------------------------------------|---------------|
| **enabled** | bool | Turns automatic rotation on or off. | None         |
| **rotationFrequency** | string | Defines the desired interval between binding rotations.  Specify time units using "m" (minutes) or "h" (hours). Note that | "m", "h"  |
| **rotatedBindingTTL** | string | Determines how long to keep the old ServiceBinding resource after rotation and before deletion. The actual TTL may be slightly longer. Specify time units using "m" (minutes) or "h" (hours).   | "m", "h"  |   

## Rotation Process

The `credentialsRotationPolicy` is evaluated periodically during a [control loop](https://kubernetes.io/docs/concepts/architecture/controller/), which runs on every service binding update or during a full reconciliation process. This means the actual rotation occurs in the closest upcoming reconciliation loop. 

## Immediate Rotation

You can trigger an immediate rotation regardless of the configured **rotationFrequency** by adding the `services.cloud.sap.com/forceRotate: "true"` annotation to the ServiceBinding resource. This immediate rotation only works if automatic rotation is already enabled. 

> [!NOTE]
> The `credentialsRotationPolicy` has no control over the validity of the credentials. The content and expiration time of the credentials is determined by the service you're using.

The following example shows the configuration of a ServiceBinding resource to rotate credentials every 25 days (600 hours) and keep the old ServiceBinding resource for 2 days (48 hours) before deleting it:

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

Automatic credential rotation cannot be enabled for a backup ServiceBinding (named: original-binding-name(variable)-guid(variable)) which is marked with the `services.cloud.sap.com/stale` label.
This backup service binding is created during the credentials rotation process to facilitate the process.
