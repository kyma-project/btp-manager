# Use BTP Manager to Manage SAP BTP Service Operator 

## Create and Install Secret

To create a real BTP Manager Secret, follow these steps:
1. Clone the `btp-manager` repository to your local file system.
2. Create ServiceBinding to obtain the access credentials to the ServiceInstance as described in points 2 of the [Setup](https://github.com/SAP/sap-btp-service-operator#setup) section in the SAP BTP service operator documentation.
3. Copy and save the access credentials into your `hack/creds.json` file in the cloned `btp-manager` repository.
4. Call [`create-secret-file.sh`](https://github.com/kyma-project/btp-manager/blob/main/hack/create-secret-file.sh). 
5. Apply the Secret in your cluster. 

   > **CAUTION:** The Secret already contains the required label: `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. Without this label, the Secret would not be visible to BTP Manager.

   ```sh
   ./hack/create-secret-file.sh
   kubectl apply -f hack/operator-secret.yaml
   ```
  

To check the `BtpOperator` custom resource (CR) status, run the following command:
```sh
kubectl get btpoperators btpoperator
```

The expected result is:
```
NAME                 STATE
btpoperator          Ready
```

## Deploy an SAP BTP Service in Your Kyma Cluster

After successfully installing your Secret, you can create a ServiceInstance and a ServiceBinding.

> **NOTE:** This section provides a real example with the real `auditlog-api` service. Use your real Secret to successfully complete the procedure.

1. To create a ServiceInstance, run the following script:

    ```yaml
    kubectl create -f - <<EOF
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceInstance
    metadata:
      name: btp-audit-log-instance
      namespace: default
    spec:
      serviceOfferingName: auditlog-api
      servicePlanName: default
      externalName: btp-audit-log-instance
    EOF
    ```

    >**TIP:** You can find values for the **serviceOfferingName** and **servicePlanName** parameters in the Service Marketplace of the SAP BTP cockpit. Click on the service's tile and find **name** and **Plan** respectively. The value of the **externalName** parameter must be unique.

2. To check the output, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com btp-audit-log-instance -o yaml
    ```

    You see the status `created` and the message `ServiceInstance provisioned successfully`.

3. To create a ServiceBinding, run this script:

    ```yaml
    kubectl create -f - <<EOF
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceBinding
    metadata:
      name: btp-audit-log-binding
      namespace: default
    spec:
      serviceInstanceName: btp-audit-log-instance
      externalName: btp-audit-log-binding
      secretName: btp-audit-log-binding
    EOF
    ```

4. To check the output, run:

    ```bash
    kubectl get servicebindings.services.cloud.sap.com btp-audit-log-binding -o yaml
    ```

    You see the status `created` and the message `ServiceBinding provisioned successfully`.

5. Now use a given service in your Kyma cluster. To see credentials, run:

    ```bash
    kubectl get secret btp-audit-log-binding -o yaml
    ```

6. Clean up your resources by running the following command:

    ```bash
    kubectl delete servicebindings.services.cloud.sap.com btp-audit-log-binding
    kubectl delete serviceinstances.services.cloud.sap.com btp-audit-log-instance
    ```
    To remove the Secret, use the following command:
    ```bash
    kubectl delete -f hack/operator-secret.yaml
    ```

## Create a ServiceInstance with a Custom Secret

To create a ServiceInstance, you must use the **btpAccessCredentialsSecret** field in the spec of the ServiceInstance. In it, you pass the Secret from the `kyma-system` namespace. The Secret is used to create your ServiceInstance. You can use different Secrets for different ServiceInstances.
> **CAUTION:** Once you set a Secret name in the ServiceInstance, you cannot change it in the future.

Adding the access credentials of the SAP BTP Service Manager Instance in your ServiceInstance results in displaying the subaccount ID to which the instance belongs in the status **subaccountID** field.

To create a ServiceInstance with a custom Secret, follow these steps:

1. Get the access credentials of the SAP BTP Service Manager Instance with a `service-operator-access` plan from its service binding. Copy them from the BTP cockpit as a JSON. 

2. Go to `hack/creds.json` and insert the credentials there.

3. To generate a Secret, call the `create-secret-file.sh` script with the **operator** option as the first parameter and **your-secret-name** as the second parameter.

   ```sh
   ./hack/create-secret-file.sh operator test-secret
    kubectl apply -f btp-access-credentials-secret.yaml
   ```

4. When you have your Secret, create your ServiceInstance with the **btpAccessCredentialsSecret** field in spec pointing to the newly created `test-secret` Secret and with other parameters as needed.

Here is an example of a ServiceInstance which you can apply:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceInstance
metadata:
  name: test-service-instance
  namespace: default
spec:
  serviceOfferingName: xsuaa
  servicePlanName: application
  btpAccessCredentialsSecret: test-secret
```