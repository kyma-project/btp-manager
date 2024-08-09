# Create a ServiceInstance with a Custom Secret

To create a ServiceInstance, you must use the **btpAccessCredentialsSecret** field in the spec of the ServiceInstance. In it, you pass the Secret from the `kyma-system` namespace. The Secret is used to create your ServiceInstance. You can use different Secrets for different ServiceInstances.

> [!WARNING] 
> Once you set a Secret name in the ServiceInstance, you cannot change it in the future.

Adding the access credentials of the SAP BTP Service Manager Instance in your ServiceInstance results in displaying the subaccount ID to which the instance belongs in the status **subaccountID** field.

To create a ServiceInstance with a custom Secret, follow these steps:

1. Get the access credentials of the SAP BTP Service Manager Instance with the `service-operator-access` plan from its ServiceBinding. Copy them from the BTP cockpit as a JSON. 

2. Create the `creds.json` file in your working directory and save the credentials there.

3. In the same working directory, generate a Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **your-secret-name** as the second parameter.

   ```sh
    curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s operator 'test-secret'
    kubectl apply -f btp-access-credentials-secret.yaml
   ```

4. When you have the Secret, create your ServiceInstance with the **btpAccessCredentialsSecret** field in spec pointing to the newly created `test-secret` Secret and with other parameters as needed.

   Here is an example of a ServiceInstance which you can apply:

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

5. To verify that the ServiceInstance has been created successfully, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com test-service-instance -o yaml
    ```

    You see the status `created` and the message `ServiceInstance provisioned successfully`.
    You also see the `test-secret` value in the **btpAccessCredentialsSecret** spec field.
    In the status section,  the **subaccountId** field must not be empty.

6. Clean up your resources by running the following command:

    ```bash
    kubectl delete serviceinstances.services.cloud.sap.com test-service-instance
    ```

    If you are not using the `test-secret` Secret for other ServiceInstances, you can delete it with this command:

    ```bash
    kubectl delete secret test-secret -n kyma-system
    ```
