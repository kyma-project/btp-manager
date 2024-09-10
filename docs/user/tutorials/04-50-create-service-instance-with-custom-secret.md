# Create a Service Instance with a Custom Secret

## Context
To create a service instance, you must use the **btpAccessCredentialsSecret** field in the `spec` of the service instance. In it, you pass the Secret from the `kyma-system` namespace to create your service instance. You can use different Secrets for different service instances.

> [!WARNING] 
> Once you set a Secret name in the service instance, you cannot change it in the future.

When you add the access credentials of the SAP Service Manager instance in your service instance, check the subaccount ID to which the instance belongs in the status **subaccountID** field.

## Prerequisites
* The `sap-btp-service-operator` Secret. For instructions on creating the Secret, see [Create a Custom `sap-btp-service-operator` Secret](04-20-create-btp-service-operator-secret.md).

## Procedure

1. When you have the Secret, create your ServiceInstance with the **btpAccessCredentialsSecret** field in the `spec` pointing to the new `my-secret` Secret and with other parameters as needed.

   Here is an example of a ServiceInstance custom resource:

   ```yaml
   kubectl create -f - <<EOF
   apiVersion: services.cloud.sap.com/v1
   kind: ServiceInstance
   metadata:
     name: {SERVICE_INSTANCE_NAME}
     namespace: default
   spec:
     serviceOfferingName: xsuaa
     servicePlanName: application
     btpAccessCredentialsSecret: {YOUR_SECRET_NAME}
   EOF
   ```

## Result

To verify that your service instance has been created successfully, run the following command:

```bash
kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -o yaml
```

You see the status `Created` and the message `ServiceInstance provisioned successfully`.
You also see the `{YOUR_SECRET_NAME}` value in the **btpAccessCredentialsSecret** field of the `spec`.
In the status section, the **subaccountId** field must not be empty.
