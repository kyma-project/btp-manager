# Management of the Service Instances and Service Bindings Lifecycle

Use the SAP BTP Operator module to manage the lifecycle of service instances and service bindings in the Kyma environment.

<!--it's very similar to 04-30-deploy-service-in-cluster.md, so maybe one should be for the OS and the other for commercial customers? There's also Using SAP BTP Services in the Kyma Environment: https://help.sap.com/docs/btp/sap-business-technology-platform/using-sap-btp-services-in-kyma-environment-->

## Create a Service Instance

1.  To create an instance of an SAP BTP service, create a ServiceInstance custom resource file following this example:
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
    > [!NOTE]
    > In the **serviceOfferingName**, enter the name of the SAP BTP service you want to use. For the **servicePlanName**, use the plan of the SAP BTP service you want to use.

2.  Apply the custom resource file in your cluster to create the service instance.

    ```bash
    kubectl apply -f path/to/my-service-instance.yaml
    ```
    
3.  Check the service's status in your cluster. The expected result is `Created`.
   
    ```bash
    kubectl get serviceinstances
    NAME                  OFFERING          PLAN        STATUS    AGE
    my-service-instance   <offering>        <plan>      Created   44s
    ```

## Service Binding

A ServiceBinding custom resource allows an application to obtain access credentials for communicating with an SAP BTP service. 
These access credentials are available to applications through a Secret resource generated in your cluster.

This is an example of the ServiceBinding CR:
<!--replace sample-binding and sample-instance with my-service-instance like above?-->
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
> [!NOTE] 
> In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.

### Create a Service Binding

1.  Apply the custom resource file in your cluster to create the service binding:

    ```bash
    kubectl apply -f path/to/my-binding.yaml
    ```
    
2.  Verify that your service binding status is `Created`:

    ```bash
    kubectl get servicebindings
    NAME         INSTANCE              STATUS    AGE
    my-binding   my-service-instance   Created   16s    
    ```

3.  Check that the Secret with the name specified in the  **spec.secretName** field of the ServiceBinding custom resource is created. The Secret contains access credentials that the applicatiions require to use the service:

    ```bash
    kubectl get secrets
    NAME         TYPE     DATA   AGE
    my-secret    Opaque   5      32s
    ```

    See [Uses for Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#uses-for-secrets) to learn about different options of using the credentials from your application running in the Kubernetes cluster.


## Delete Service Instances and Service Bindings
 <!--add instructions on how to delete a cluster?? SIs and SBs???-->
To delete a service instance and service binding, use the following commands:
<!--replace sample-binding and sample-instance with my-service-instance like above?-->
```bash
kubectl delete servicebindings.services.cloud.sap.com sample-binding
kubectl delete serviceinstances.services.cloud.sap.com my-service-instance
```

When you want to delete a Kyma cluster, ensure that you first delete all the service instances and service bindings associated  with the `sap-btp-service-operator` Secret in the `kyma-system` namespace. Otherwise, the deletion of your cluster is blocked.
If you have not deleted service instances and bindings connected to your expired free tier service or trial cluster, you can still find the service binding credentials in the SAP Service Manager instance details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings. 

<!--link do preconfigured credentials or related info??-->