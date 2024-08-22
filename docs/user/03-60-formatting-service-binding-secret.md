# Formats of Service Binding Secrets

Use different attributes in your ServiceBinding resource to generate different formats of your Secret resources.

Secret resources share a common set of basic parameters that can be divided into two categories:
* Credentials returned from the service broker: They allow your application to access and consume a SAP BTP service.
* Attributes of the associated service instance: The details of the service instance itself.

However, the Secret resources can come in various formats:
* Default key-value pairs
* A JSON object
* One JSON object with credentials and service information
* Custom formats


## Key-Value Pairs

If you do not use any of the attributes, the generated `Secret` is by default in a key-value pair format. 

`ServiceBinding`

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: sample-binding
spec:
  serviceInstanceName: sample-instance
```
`Secret`

```yaml
apiVersion: v1
metadata:
  name: sample-binding
kind: Secret
stringData:
  uri: https://my-service.authentication.eu10.hana.ondemand.com
  client_id: admin
  client_secret: ********
  instance_guid: your-sample-instance-guid // The service instance ID
  instance_name: sample-instance // Taken from the service instance external_name field if set. Otherwise from metadata.name
  plan: sample-plan // The service plan name                
  type: sample-service  // The service offering name
```

## Credentials as JSON Object

To show credentials that the <!--??--> broker returns within the `Secret` resource as a JSON object, use the `secretKey` attribute in the `ServiceBinding` spec.
The value of the `secretKey` is the name of the key that stores the credentials in the JSON format:

`ServiceBinding`

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: sample-binding
spec:
  serviceInstanceName: sample-instance
  secretKey: myCredentials
```
`Secret`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sample-binding
stringData:
    myCredentials:
      uri: https://my-service.authentication.eu10.hana.ondemand.com,
      client_id: admin,
      client_secret: ********
    instance_guid: your-sample-instance-guid // The service instance ID
    instance_name: sample-binding // Taken from the service instance external_name field if set. Otherwise from metadata.name 
    plan: sample-plan // The service plan name
    type: sample-service // The service offering name
```

## Credentials and Service Info as One JSON Object

To show both credentials returned from the broker and additional `ServiceInstance` attributes as a JSON object, use the `secretRootKey` attribute in the `ServiceBinding` spec.

The value of `secretRootKey` is the name of the key that stores both credentials and `ServiceInstance` info in the JSON format.

`ServiceBinding`

```yaml
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: sample-binding
spec:
  serviceInstanceName: sample-instance
  secretRootKey: myCredentialsAndInstance
```
`Secret`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sample-binding
stringData:
    myCredentialsAndInstance:
        uri: https://my-service.authentication.eu10.hana.ondemand.com,
        client_id: admin,
        client_secret: ********,
        instance_guid: your-sample-instance-guid, // The service instance id
        instance_name: sample-instance-name, // Taken from the service instance external_name field if set. Otherwise from metadata.name
        plan: sample-instance-plan, // The service plan name
        type: sample-instance-offering, // The service offering name
```
## Custom Formats 

For additional flexibility, you can model the Secret resources according to your needs. 
To generate a custom-formatted Secret, use the **secretTemplate** attribute in the ServiceBinding spec.
This attribute expects a Go template as its value. For more information, see [Go Templates](https://pkg.go.dev/text/template).<br>
Ensure the template is in the YAML format, and it has the structure of a Kubernetes Secret. 

In the provided Secret, you can customize the `metadata` and `stringData` sections with the following options:

- `metadata`: labels and annotations
- `stringData`: customize or utilize one of the available formatting options as detailed in the [Formats of Service Binding Secrets](#formats-of-service-binding-secrets) section.


> [!NOTE]  
> If you customize `stringData`, it takes precedence over the provided pre-defined formats.

The provided templates are executed on a map with the following available attributes:

| Reference         | Description                                |                                                                          
|-----------------|--------------------------------------------|
| **instance.instance_guid** |  The service instance ID.     |
| **instance.instance_name** |  The service instance name.   |                                                
| **instance.plan**   |  The name of the service plan used to create this service instance. |  
| **instance.type**   |  The name of the associated service offering. |  
| **credentials.attributes(var)**   |  The content of the credentials depends on a service. For more details, refer to the documentation of the service you're using. |  

See two examples demonstrating the ServiceBinding and generated Secret resources. The first ServiceBinding example utilizes a custom template, while the second example combines a custom template with a predefined formatting option:

* Example of a binding with customized `metadata` and `stringData` sections:

    In this example, you specify both `Metadata` and `stringData` in the `secretTemplate`:

    `ServiceBinding`

    ```yaml
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceBinding
    metadata:
    name: sample-binding
    spec:
    serviceInstanceName: sample-instance
    secretTemplate: |
        apiVersion: v1
        kind: Secret
        metadata:
        labels:
            service_plan: {{ .instance.plan }}
        annotations:
            instance: {{ .instance.instance_name }}
        stringData:
        USERNAME: {{ .credentials.client_id }}
        PASSWORD: {{ .credentials.client_secret }}
    ```

    `Secret`

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
    labels:
        service_plan: sample-plan
    annotations:
        instance: sample-instance
    stringData:
    USERNAME: admin
    PASSWORD: ********
    ```

* Example of a binding with a customized `metadata` section and applied pre-existing formatting option for `stringData` (credentials as JSON object):

    In this example, you omit `stringData` from the `secretTemplate` and use the `secretKey` to format your `stringData` instead.

    `ServiceBinding`

    ```yaml
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceBinding
    metadata:
    name: sample-binding
    spec:
    serviceInstanceName: sample-instance
    secretKey: myCredentials
    secretTemplate: |
        apiVersion: v1
        kind: Secret
        metadata:
        labels:
            service_plan: {{ .instance.plan }}
        annotations:
            instance: {{ .instance.instance_name }}
    ```

    `Secret`

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
    labels:
        service_plan: sample-plan
    annotations:
        instance: sample-instance
    stringData:
    myCredentials:
        uri: https://my-service.authentication.eu10.hana.ondemand.com,
        client_id: admin,
        client_secret: ********
    instance_guid: your-sample-instance-guid // The service instance ID
    instance_name: sample-binding // Taken from the service instance external_name field if set. Otherwise from metadata.name
    plan: sample-plan // The service plan name
    type: sample-service // The service offering name
    ```
    