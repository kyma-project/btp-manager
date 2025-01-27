# Delete Service Bindigs and Service Instances

Delete service bindings and service instances using Kyma dashboard or kubectl.

## Context

You can only delete service instances or service bindings created in Kyma using Kyma dashboard or kubectl. You can't perform these operations using the SAP BTP cockpit. To delete a service instance, first delete its service bindings.

> [!WARNING]
> Once you delete your service instances and service bindings, you cannot revert the operation.

Use either Kyma dashboard or kubectl to delete a service binding or a service instance.

## Procedure

<!-- tabs:start -->
#### **Kyma Dashboard**

1. In the **Namespaces** view, go to the namespace you want to delete a service binding/instance from.
2. Go to **Service Management** -> **Service Bindings**/**Service Instances**.
3. Delete the service binding/instance.

#### **kubectl**

To delete a service binding, run:

```bash
kubectl delete servicebindings.services.cloud.sap.com {BINDING_NAME}
```

To delete a service instance, run:

```bash
kubectl delete serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME}
```
<!-- tabs:end -->

## Next Steps

If you want to delete your Kyma cluster, you may have to first delete your service instances and bindings from it.

You can't delete your Kyma cluster if any non-deleted service instances in it use the credentials from the SAP Service Manager resources created automatically, as described in [Preconfigured Credentials and Access](03-10-preconfigured-secret.md#credentials). In this case, the existing service instances block the cluster's deletion. Delete your service instances and bindings in Kyma dashboard before you attempt to delete the cluster from the SAP BTP cockpit.

You can delete your Kyma cluster even if your service instances still exist, provided they all use credentials of SAP Service Manager service instances other than the one created automatically, as described in [Preconfigured Credentials and Access](03-10-preconfigured-secret.md#credentials). In this case, the non-deleted service instances do not block the cluster's deletion. 

If you have not deleted service instances and bindings connected to your expired free tier service, you can still find the service binding credentials in the SAP Service Manager instance details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings.
