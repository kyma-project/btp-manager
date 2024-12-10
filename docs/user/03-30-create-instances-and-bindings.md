# Create  Service Instances and Service Bindings

To use an SAP BTP service in your Kyma cluster, create its service instance and service binding using Kyma dashboard or kubectl.

## Prerequisites

* The [SAP BTP Operator module](README.md) is added. For instructions on adding modules, see [Add and Delete a Kyma Module](https://help.sap.com/docs/btp/sap-business-technology-platform/enable-and-disable-kyma-module).
* For CLI interactions: [kubectl](https://kubernetes.io/docs/tasks/tools/) v1.17 or higher.
* You know the service offering name and service plan name for the SAP BTP service you want to connect to your Kyma cluster.
  To find the service and service plan names, in the SAP BTP cockpit, go to **Services**->**Service Marketplace**. Click on the service tile and find its **name** and **Plan**.

> [!NOTE]
> You can use [SAP BTP kubectl plugin](https://github.com/SAP/sap-btp-service-operator#sap-btp-kubectl-plugin-experimental) to get the available services in your SAP BTP account by using the access credentials stored in the cluster. However, the plugin is still experimental.

## Create a Service Instance

### Context

To create a service instance, use either Kyma dashboard or kubectl.

<!-- tabs:start -->
#### Use Kyma Dashboard

1. In the **Namespaces** view, go to the namespace you want to work in.
2. Go to **Service Management** -> **Service Instances**.
3. Provide the required service details and create a service instance.<br>
   You see the status `PROVISIONED`.

#### Use kubectl

1.  To create a ServiceInstance custom resource (CR), follow this example:

    ```yaml
        kubectl create -f - <<EOF 
        apiVersion: services.cloud.sap.com/v1
        kind: ServiceInstance
        metadata:
            name: {SERVICE_INSTANCE_NAME}
            namespace: {NAMESPACE} 
        spec:
            serviceOfferingName: {SERVICE_OFFERING_NAME}
            servicePlanName: {SERVICE_PLAN_NAME}
            externalName: {SERVICE_INSTANCE_NAME}
            parameters:
              key1: val1
              key2: val2
        EOF
    ```
      In the **serviceOfferingName** and  **servicePlanName** fields, enter the name of the SAP BTP service you want to use and the service plan respectively.
    
2.  To check the service's status in your cluster, run:
   
    ```bash
    kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -n {NAMESPACE}
    ```

    You get an output similar to this one:

    ```
    NAME                      OFFERING                  PLAN                  STATUS    AGE
    {SERVICE_INSTANCE_NAME}   {SERVICE_OFFERING_NAME}   {SERVICE_PLAN_NAME}   Created   44s
    ```
<!-- tabs:end -->

## Create a Service Binding

### Context

With a ServiceBinding custom resource (CR), your application can get access credentials for communicating with an SAP BTP service.
These access credentials are available to applications through a Secret resource generated in your cluster.

To create a service binding, use either Kyma dashboard or kubectl.

### Procedure

<!-- tabs:start -->
#### Use Kyma Dashboard

1. In the **Namespaces** view, go to the namespace you want to work in.
2. Go to **Service Management** -> **Service Bindings**.
3. Choose your service instance name from the dropdown list and create a service binding.<br>
   You see the status `PROVISIONED`.

#### Use kubectl

1. To create a ServiceBinding CR, follow this example:

      ```yaml
      kubectl create -f - <<EOF
      apiVersion: services.cloud.sap.com/v1
      kind: ServiceBinding
      metadata:
        name: {BINDING_NAME}
      spec:
        serviceInstanceName: {SERVICE_INSTANCE_NAME}
        externalName: {EXTERNAL_NAME}
        secretName: {SECRET_NAME}
        parameters:
          key1: val1
          key2: val2   
      EOF        
      ```

    > [!NOTE]
    > In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.
    
2.  To check your service binding status, run:

    ```bash
    kubectl get servicebindings {BINDING_NAME} -n {NAMESPACE}
    ```

    You see the staus `Created`.

3.  Verify the Secret is created with the name specified in the  **spec.secretName** field of the ServiceBinding CR. The Secret contains access credentials that the applications need to use the service:

    ```bash
    kubectl get secrets {SECRET_NAME} -n {NAMESPACE}
    ```
    You see the same Secret name as in the spec.secretName field of the ServiceBinding CR.
<!-- tabs:end -->

### Results

You can use a given service in your Kyma cluster.
