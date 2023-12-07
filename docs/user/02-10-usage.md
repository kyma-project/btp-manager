# Use BTP Manager to Manage SAP BTP Service Operator 

## Create and Install Secret

To create a real BTP Manager Secret, follow these steps:
1. Clone the `btp-manager` repository to your local file system.
2. Create ServiceBinding to obtain the access credentials to the ServiceInstance as described in points 2 of the [Setup](https://github.com/SAP/sap-btp-service-operator#setup) section in the SAP BTP service operator documentation.
3. Copy and save the access credentials into your `hack/creds.json` file in the cloned `btp-manager` repository.
4. Call [`create-secret-file.sh`](https://github.com/kyma-project/btp-manager/blob/main/hack/create-secret-file.sh). 
5. Apply the Secret in your cluster. 
 
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
