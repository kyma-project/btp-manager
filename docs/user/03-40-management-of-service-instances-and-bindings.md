# Manage the Lifecycle of Service Instances and Service Bindings

Use the SAP BTP Operator module to manage the lifecycle of service instances and service bindings in the Kyma environment.

<!--it's very similar to 04-30-deploy-service-in-cluster.md, so maybe one should be for the OS and the other for commercial customers? There's also Using SAP BTP Services in the Kyma Environment: https://help.sap.com/docs/btp/sap-business-technology-platform/using-sap-btp-services-in-kyma-environment-->

## Create a Service Instance

1.  To create an instance of an SAP BTP service, create a ServiceInstance custom resource file following this example:

    ```yaml
        apiVersion: services.cloud.sap.com/v1
        kind: ServiceInstance
        metadata:
            name: {INSTANCE_NAME}
        spec:
            serviceOfferingName: {NAME_FROM_SERVICE_MARKETPLACE}
            servicePlanName: {PLAN_FROM_SERVICE_MARKETPLACE}
            externalName: {INSTANCE_NAME}
            parameters:
              key1: val1
              key2: val2
    ```
<!-- in the HP, there's also namespace: {NAMESPACE} under the name; shouldn't it be added here?-->
1.  Apply the custom resource file in your cluster to create the service instance.

    ```bash
    kubectl apply -f path/to/my-service-instance.yaml
    ```
<!-- how to add {INSTANCE_NAME} here? maybe: ??-->
```bash
kubectl apply -f path/to/{INSTANCE_NAME}.yaml
```
<!-- REMOVE ONE OF THE OPTIONS!!!!!!-->
1.  Check the service's status in your cluster. The expected result is **Created**.

    ```bash
    kubectl get serviceinstances
    NAME                  OFFERING          PLAN        STATUS    AGE
    {INSTANCE_NAME}       <offering>        <plan>      Created   44s
    ```

## Service Binding

To allow an application to obtain access credentials to communicate with a SAP BTP service, create a ServiceBinding custom resource. In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.

These access credentials are available to applications through a `Secret` resource generated in your cluster.

See the structure of the ServiceBinding CR:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: {BINDING_NAME}
spec:
  serviceInstanceName: {INSTANCE_NAME}
  externalName: {BINDING_NAME}
  secretName: my-secret
  parameters:
    key1: val1
    key2: val2      
```
<!--WHAT ABOUT secretName: my secret? In HP it's secretName:{BINDING_NAME}; should it be the same here???-->
### Create a Service Binding

1.  Apply the custom resource file in your cluster to create the service binding:

    ```bash
    kubectl apply -f path/to/my-binding.yaml
    ```
    <!--OR:???-->
    ```bash
    kubectl apply -f path/to/{BINDING_NAME}.yaml
    ```
    <!-- REMOVE ONE OF THE OPTIONS!!!!!!-->
2.  Verify that your service binding status is **Created** before you proceed:

    ```bash
    kubectl get servicebindings
    NAME             INSTANCE              STATUS    AGE
    {BINDING_NAME}   {INSTANCE_NAME}       Created   16s
    
    ```

3.  Check that the Secret with the name as specified in the  **spec.secretName** field of the ServiceBinding custom resource is created. The Secret contains access credentials needed for the applicatiions to use the service:

    ```bash
    kubectl get secrets
    NAME         TYPE     DATA   AGE
    my-secret   Opaque   5      32s
    ```
    <!--WHAT ABOUT secretName: my secret?-->
    See [Uses for Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#uses-for-secrets) to learn about different options on how to use the credentials from your application running in the Kubernetes cluster.


### Delete Service Instances and Service Bindings

When you delete a Kyma cluster, ensure that you first delete all the service instances and service bindings associated  with the `sap-btp-service-operator` Secret in the `kyma-system` namespace. Otherwise, the deletion of your cluster is blocked.
If your free tier service or trial cluster expires and you do not delete the service instances and bindings connected to it, you can still find the service binding credentials in the SAP Service Manager instance details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings. <!--CreatedBy
:
CIS-->

 <!--add instructions on how to delete a cluster?? SIs and SBs???-->
To delete a service instance and service binding, use the following commands:

```bash
kubectl delete servicebindings.services.cloud.sap.com {BINDING_NAME}
kubectl delete serviceinstances.services.cloud.sap.com {INSTANCE_NAME}
```
<!--link do preconfigured credentials or related info??-->