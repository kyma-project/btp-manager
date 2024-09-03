# Create an SAP BTP Service Instance in Your Kyma Cluster

After successfully installing your Secret, create a service instance and a service binding.

> [!NOTE] 
> This section provides a real example with the real `xsuaa` service. Use your real Secret to complete the procedure successfully.

## Procedure

1. To create a service instance, run the following script:

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
      externalName: my-service-instance
    EOF
    ```

   > [!TIP] 
   > To find values for the **serviceOfferingName** and **servicePlanName** parameters, go to the SAP BTP cockpit > **Service Marketplace**, select the service's tile, and find the **name** and **Plan**. The value of the **externalName** parameter must be unique.

2. To check the output, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com my-service-instance -o yaml
    ```

    You see the status `Created` and the message `ServiceInstance provisioned successfully`.

3. To create a service binding, run this script:

    ```yaml
    kubectl create -f - <<EOF
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceBinding
    metadata:
      name: my-service-binding
      namespace: default
    spec:
      serviceInstanceName: my-service-instance
      externalName: my-service-binding
      secretName: my-service-binding
    EOF
    ```

4. To check the output, run:

    ```bash
    kubectl get servicebindings.services.cloud.sap.com my-service-binding -o yaml
    ```

    You see the status `Created` and the message `ServiceBinding provisioned successfully`.

5. Now, use a given service in your Kyma cluster. To see credentials, run:

    ```bash
    kubectl get secret my-service-binding -o yaml
    ```

## Next Steps

To clean up your resources, run the following command:

```bash
kubectl delete servicebindings.services.cloud.sap.com my-service-binding
kubectl delete serviceinstances.services.cloud.sap.com my-service-instance
```
