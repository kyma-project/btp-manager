# Management of the Service Instances and Service Bindings Lifecycle

Use the SAP BTP Operator module to manage the lifecycle of service instances and service bindings in the Kyma environment.

## Create a Service Instance

1.  To create an instance of an SAP BTP service, create a ServiceInstance custom resource (CR) file following this example: <!--add line 9 and skip step 2?; any namespace or kyma-system; REMOVE line 23-->
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
      In the **serviceOfferingName** and  **servicePlanName** fields, enter the name of the SAP BTP service you want to use and the service's plan respectively.
    
2.  Check the service's status in your cluster. The expected result is `Created`.
   
    ```bash
    kubectl get serviceinstances -n {SERVICE-INSTANCE-NAME}
    NAME                      OFFERING                    PLAN                     STATUS    AGE
    {SERVICE-INSTANCE-NAME}   {SERVICE_OFFERING_NAME}     {SERVICE_PLAN_NAME}      Created   44s
    ```

## Create a Service Binding

With a ServiceBinding CR, your application can get access credentials for communicating with an SAP BTP service. 
These access credentials are available to applications through a Secret resource generated in your cluster.

1. Create a ServiceBinding CR based on the following example:<!--externalName? secretName?-->

      ```yaml
      kubectl create -f - <<EOF
      apiVersion: services.cloud.sap.com/v1
      kind: ServiceBinding
      metadata:
        name: {BINDING_NAME}
      spec:
        serviceInstanceName: {SERVICE_INSTANCE_NAME}
        externalName: {BINDING_NAME}
        secretName: {BINDING_NAME}
        parameters:
          key1: val1
          key2: val2   
      EOF        
      ```

    In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.
    
2.  Verify that your service binding status is `Created`:

    ```bash
    kubectl get servicebindings
    NAME              INSTANCE                  STATUS    AGE
    {BINDING_NAME}    {SERVICE_INSTANCE_NAME}   Created   16s    
    ```

3.  Verify the Secret is created with the name specified in the  **spec.secretName** field of the ServiceBinding CR. The Secret contains access credentials that the applications need to use the service:

    ```bash
    kubectl get secrets
    NAME              TYPE     DATA   AGE
    {BINDING_NAME}    Opaque   5      32s
    ```

    To learn about different options for using the credentials from your application running in the Kubernetes cluster, see [Uses for Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#uses-for-secrets).

## Pass Parameters

To set input parameters, go to the `spec` of the ServiceInstance or ServiceBinding resource, and use both or one of the following fields:
* **parameters**: Specifies a set of properties sent to the service broker.
  The specified data is passed to the service broker without any modifications - aside from converting it to JSON for transmission to the broker if the `spec` field is specified as YAML.
  All valid YAML or JSON constructs are supported. 
  > [!NOTE] 
  > Only one parameter field per `spec` can be specified.
* **parametersFrom**: Specifies which Secret, together with the key in it, to include in the set of parameters sent to the service broker.
  The key contains a `string` that represents the JSON. The **parametersFrom** field is a list that supports multiple sources referenced per `spec`.

If you specified multiple sources in the **parameters** and **parametersFrom** fields, the final payload results from merging all of them at the top level.
If there are any duplicate properties defined at the top level, the specification is considered to be invalid. 
The further processing of the ServiceInstance or ServiceBinding resource stops with the status `Error`.

See the following example of the `spec` format in YAML:
```yaml
spec:
  ...
  parameters:
    name: value
  parametersFrom:
    - secretKeyRef:
        name: {SECRET_NAME}
        key: secret-parameter
```

See the following example of the `spec` format in JSON:

```json
{
  "spec": {
    "parameters": {
      "name": "value"
    },
    "parametersFrom": {
      "secretKeyRef": {
        "name": "{SECRET_NAME}",
        "key": "secret-parameter"
      }
    }
  } 
}
```

See the exampple of a Secret with the key named **secret-parameter**:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
type: Opaque
stringData:
  secret-parameter:
    '{
      "password": "password"
    }'
```

See the example of the final JSON payload sent to the service broker:
```json
{
  "name": "value",
  "password": "password"
}
```

To list multiple parameters in the Secret, separate key-value pairs with commas. See the following example:

```yaml
secret-parameter:
  '{
    "password": "password",
    "key2": "value2",
    "key3": "value3"
  }'
```


## Delete Service Bindigs and Service Instance

You can't delete service instances or service bindings created in Kyma using the SAP BTP cockpit. You can only perform these operations using Kyma dashboard or kubectl.

> [!WARNING]
> Once you delete your service instance and service bindings, you cannot revert the operation.

### Delete Resources Using Kyma Dashboard

> [!TIP]
> To successfully delete a service instance, first delete its service binding(s).

1. In the **Namespace** view, go to **Service Management** -> **Service Bindings**/**Service Instances**.
2. Use the trashbin icon to delete the service binding/instance.

### Delete Resources Using kubectl

To delete a service binding, run:

```bash
kubectl delete serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_BINDING}
```

To delete a service instance, run:

```bash
kubectl delete serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE}
```

> [!NOTE]
> If you haven't deleted all the service instances and service bindings associated  with the `sap-btp-service-operator` Secret in the `kyma-system` namespace, you can't delete your Kyma cluster from the SAP BTP cockpit. To delete the remaining service instances and service bindings, go to Kyma dashboard.<br>
> If you have not deleted service instances and bindings connected to your expired free tier service or trial cluster, you can still find the service binding credentials in the SAP Service Manager instance details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings.
