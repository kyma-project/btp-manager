# Manage the Lifecycle of Service Instances and Service Bindings

Use the SAP BTP Operator module to manage the lifecycle of service instances and service bindings.

<!--it's very similar to 04-30-deploy-service-in-cluster.md, so maybe one should be for the OS and the other for commercial customers? There's also Using SAP BTP Services in the Kyma Environment: https://help.sap.com/docs/btp/sap-business-technology-platform/using-sap-btp-services-in-kyma-environment-->

## Create a Service Instance

1.  To create an instance of a SAP BTP service, create a ServiceInstance custom resource file:

    ```yaml
        apiVersion: services.cloud.sap.com/v1
        kind: ServiceInstance
        metadata:
            name: my-service-instance
        spec:
            serviceOfferingName: sample-service
            servicePlanName: sample-plan
            externalName: my-service-btp-name
            parameters:
            key1: val1
            key2: val2
    ```

1.  Apply the custom resource file in your cluster to create the service instance.

    ```bash
    kubectl apply -f path/to/my-service-instance.yaml
    ```

2.  Check the service's status in your cluster. The expected result is **Created**.

    ```bash
    kubectl get serviceinstances
    NAME                  OFFERING          PLAN        STATUS    AGE
    my-service-instance   <offering>        <plan>      Created   44s
    ```

## Service Binding

To allow an application to obtain access credentials to communicate with a SAP BTP service, create a ServiceBinding custom resource. In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.

These access credentials are available to applications through a `Secret` resource generated in your cluster.

See the structure of the ServiceBinding CR:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: sample-binding
spec:
  serviceInstanceName: sample-instance
  externalName: my-binding-external
  secretName: my-secret
  parameters:
    key1: val1
    key2: val2      
```

### Create a Service Binding

1.  Apply the custom resource file in your cluster to create the service binding:

    ```bash
    kubectl apply -f path/to/my-binding.yaml
    ```

2.  Verify that your service binding status is **Created** before you proceed:

    ```bash
    kubectl get servicebindings
    NAME         INSTANCE              STATUS    AGE
    my-binding   my-service-instance   Created   16s
    
    ```

3.  Check that the Secret with the name as specified in the  **spec.secretName** field of the ServiceBinding custom resource is created. The Secret contains access credentials needed for the applicatiions to use the service:

    ```bash
    kubectl get secrets
    NAME         TYPE     DATA   AGE
    my-secret   Opaque   5      32s
    ```
    
    See [Using Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets) to learn about different options on how to use the credentials from your application running in the Kubernetes cluster.

### Update <!-- if relevant-->

### Delete Service Instances

When you delete a Kyma cluster, ensure that you first delete all the associated service instances and service bindings using Kyma dashboard. Otherwise, the deletion of your cluster is blocked.
If your free tier service or trial cluster expired and you did not delete the service instances and bindings connected to it, you can still find the cluster credentials in the Service Manager details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings. <!--how??-->