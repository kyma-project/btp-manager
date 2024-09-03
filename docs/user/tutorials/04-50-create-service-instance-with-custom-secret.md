# Create a Service Instance with a Custom Secret

## Context
To create a service instance, you must use the **btpAccessCredentialsSecret** field in the `spec` of the service instance. In it, you pass the Secret from the `kyma-system` namespace to create your service instance. You can use different Secrets for different service instances.

> [!WARNING] 
> Once you set a Secret name in the service instance, you cannot change it in the future.

When you add the access credentials of the SAP Service Manager instance in your service instance, you see the subaccount ID to which the instance belongs in the status **subaccountID** field.

> [!TIP]
> For instructions on how to create a SAP BTP service operator Secret, see [Create a Custom SAP BTP Service Operator Secret](04-20-create-btp-service-operator-secret.md).

## Procedure

1. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its service binding. Copy them as a JSON from the BTP cockpit. 

2. Create the `creds.json` file in your working directory and save the credentials there. <!--is this step necessary for the managed offering; if not, what's the next step?-->

3. When you have the Secret, create your ServiceInstance with the **btpAccessCredentialsSecret** field in the `spec` pointing to the new `my-secret` Secret and with other parameters as needed.

   Here is an example of a ServiceInstance custom resource:

   ```yaml
   kubectl create -f - <<EOF
   apiVersion: services.cloud.sap.com/v1
   kind: ServiceInstance
   metadata:
     name: my-service-instance
     namespace: default
   spec:
     serviceOfferingName: xsuaa
     servicePlanName: application
     btpAccessCredentialsSecret: my-secret
   EOF
   ```

## Result

To verify that your service instance has been created successfully, run the following command:

```bash
kubectl get serviceinstances.services.cloud.sap.com my-service-instance -o yaml
```

You see the status `Created` and the message `ServiceInstance provisioned successfully`.
You also see the `my-secret` value in the **btpAccessCredentialsSecret** field of the `spec`.
In the status section, the **subaccountId** field must not be empty.

## Next Steps

To clean up your resources, run the following command:

```bash
kubectl delete serviceinstances.services.cloud.sap.com my-service-instance
```

If you do not intend to use the `my-secret` Secret for other service instances, delete it with this command:

```bash
kubectl delete secret my-secret -n kyma-system
```
