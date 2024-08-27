# Create a Service Instance with a Custom Secret
<!--not ready?-->
To create a service instance, you must use the **btpAccessCredentialsSecret** field in the spec of the service instance. In it, you pass the Secret from the `kyma-system` namespace to create your service instance. You can use different Secrets for different service instances.

> [!WARNING] 
> Once you set a Secret name in the service instance, you cannot change it in the future.

Adding the access credentials of the SAP Service Manager instance in your service instance results in displaying the subaccount ID to which the instance belongs in the status **subaccountID** field.

> [!TIP]
> For instructions on how to create a SAP BTP service operator Secret, see the [dedicated tutorial](04-10-create-btp-manager-secret.md).

To create a service instance with a custom Secret, follow these steps:

1. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its service binding. Copy them from the BTP cockpit as a JSON. 

2. Create the `creds.json` file in your working directory and save the credentials there. <!--is this step necessary for the managed offering; if not, what's the next step?-->

3. When you have the Secret, create your ServiceInstance with the **btpAccessCredentialsSecret** field in spec pointing to the newly created `test-secret` Secret and with other parameters as needed.

   Here is an example of a ServiceInstance custom resource:

   ```yaml
   kubectl create -f - <<EOF
   apiVersion: services.cloud.sap.com/v1
   kind: ServiceInstance
   metadata:
     name: test-service-instance
     namespace: default
   spec:
     serviceOfferingName: xsuaa
     servicePlanName: application
     btpAccessCredentialsSecret: test-secret
   EOF
   ```

4. To verify that your service instance has been created successfully, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com test-service-instance -o yaml
    ```

    You see the status `Created` and the message `ServiceInstance provisioned successfully`.
    You also see the `test-secret` value in the **btpAccessCredentialsSecret** spec field.
    In the status section, the **subaccountId** field must not be empty.

5. To clean up your resources, run the following command:

    ```bash
    kubectl delete serviceinstances.services.cloud.sap.com test-service-instance
    ```

    If you do not intend to use the `test-secret` Secret for other service instances, delete it with this command:

    ```bash
    kubectl delete secret test-secret -n kyma-system
    ```
