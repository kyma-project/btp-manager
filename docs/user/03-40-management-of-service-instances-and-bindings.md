# Management of the Service Instances and Service Bindings Lifecycle

Use the SAP BTP Operator module to manage the lifecycle of service instances and service bindings in the Kyma environment.

## Create a Service Instance

1.  To create an instance of an SAP BTP service, create a ServiceInstance custom resource (CR) file following this example:
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
    > In the **serviceOfferingName** and  **servicePlanName** fields, enter the name of the SAP BTP service you want to use and the service's plan respectively.

2.  Apply the CR file in your cluster to create the service instance.

    ```bash
    kubectl apply -f path/to/my-service-instance.yaml
    ```
    
3.  Check the service's status in your cluster. The expected result is `Created`.
   
    ```bash
    kubectl get serviceinstances
    NAME                  OFFERING          PLAN             STATUS    AGE
    my-service-instance   sample-service    sample-plan      Created   44s
    ```

## Service Binding

A ServiceBinding CR allows an application to obtain access credentials for communicating with an SAP BTP service. 
These access credentials are available to applications through a Secret resource generated in your cluster.

The following is an example of the ServiceBinding CR:

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: my-binding
spec:
  serviceInstanceName: my-service-instance
  externalName: my-binding-external
  secretName: my-secret
  parameters:
    key1: val1
    key2: val2           
```
> [!NOTE] 
> In the **serviceInstanceName** field of the ServiceBinding, enter the name of the ServiceInstance resource you previously created.

### Create a Service Binding

1.  Apply the CR file in your cluster to create the service binding:

    ```bash
    kubectl apply -f path/to/my-binding.yaml
    ```
    
2.  Verify that your service binding status is `Created`:

    ```bash
    kubectl get servicebindings
    NAME         INSTANCE              STATUS    AGE
    my-binding   my-service-instance   Created   16s    
    ```

3.  Verify the Secret is created with the name specified in the  **spec.secretName** field of the ServiceBinding CR. The Secret contains access credentials that the applications require to use the service:

    ```bash
    kubectl get secrets
    NAME         TYPE     DATA   AGE
    my-secret    Opaque   5      32s
    ```

    See [Uses for Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#uses-for-secrets) to learn about different options for using the credentials from your application running in the Kubernetes cluster.

## Passing Parameters

To set input parameters, go to the spec of the ServiceInstance or ServiceBinding resource, and use both or one of the following fields:
* **parameters**: Specifies a set of properties sent to the service broker.
  The specified data is passed to the service broker without any modifications - aside from converting it to JSON for transmission to the broker if the `spec` field is specified as YAML.
  All valid YAML or JSON constructs are supported. 
  > [!NOTE] 
  > Only one parameter field per `spec` can be specified.
* **parametersFrom**: Specifies which Secret, together with the key in it, to include in the set of parameters sent to the service broker.
  The key contains a `string` that represents the JSON. The **parametersFrom** field is a list that supports multiple sources referenced per `spec`.

If multiple sources in the **parameters** and **parametersFrom** fields are specified, the final payload results from merging all of them at the top level.
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
        name: my-secret
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
        "name": "my-secret",
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
  name: my-secret
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


## Delete Service Instances and Service Bindings

To delete a service instance and service binding, use commands such as:

```bash
kubectl delete servicebindings.services.cloud.sap.com my-service-binding
kubectl delete serviceinstances.services.cloud.sap.com my-service-instance
```

To delete a Kyma cluster, click on **Disable Kyma** in the SAP BTP cockpit. If you haven't deleted all the service instances and service bindings associated  with the `sap-btp-service-operator` Secret in the `kyma-system` namespace, you get the message that the deletion of your cluster is blocked. Go to Kyma dashboard to delete the remaining service instances and service bindings.<br>If you have not deleted service instances and bindings connected to your expired free tier service or trial cluster, you can still find the service binding credentials in the SAP Service Manager instance details in the SAP BTP cockpit. Use them to delete the leftover service instances and bindings.
