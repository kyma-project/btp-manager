# Deploy an SAP BTP Service in Your Kyma Cluster

After successfully installing your Secret, create a service instance and a service binding.

> [!NOTE] 
> This section provides a real example with the real `xsuaa` service. Use your real Secret to successfully complete the procedure.

1. To create a service instance, run the following script:

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
      externalName: test-service-instance
    EOF
    ```

   > [!TIP] 
   > You can find values for the **serviceOfferingName** and **servicePlanName** parameters in the Service Marketplace of the SAP BTP cockpit. Click on the service's tile and find **name** and **Plan** respectively. The value of the **externalName** parameter must be unique.

2. To check the output, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com test-service-instance -o yaml
    ```

    You see the status `Created` and the message `ServiceInstance provisioned successfully`.

3. To create a service binding, run this script:

    ```yaml
    kubectl create -f - <<EOF
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceBinding
    metadata:
      name: test-service-binding
      namespace: default
    spec:
      serviceInstanceName: test-service-instance
      externalName: test-service-binding
      secretName: test-service-binding
    EOF
    ```

4. To check the output, run:

    ```bash
    kubectl get servicebindings.services.cloud.sap.com test-service-binding -o yaml
    ```

    You see the status `Created` and the message `ServiceBinding provisioned successfully`.

5. Now, use a given service in your Kyma cluster. To see credentials, run:

    ```bash
    kubectl get secret test-service-binding -o yaml
    ```

6. To clean up your resources, run the following command:

    ```bash
    kubectl delete servicebindings.services.cloud.sap.com test-service-binding
    kubectl delete serviceinstances.services.cloud.sap.com test-service-instance
    ```
