# Formats of Service Binding Secrets

Use different attributes in your ServiceBinding resource to generate different formats of your Secret resources.

## Overview

Secret resources share a common set of basic parameters that can be divided into two categories:
* Credentials returned from the service broker that allow your application to access and consume an SAP BTP service.
* Attributes of the associated service instance: The details of the service instance itself.

However, the Secret resources can come in various formats:
* Default key-value pairs
* A JSON object
* One JSON object with credentials and service information
* Custom formats


## Key-Value Pairs

If you do not use any of the attributes, the generated Secret is by default in the key-value pair format, as in the following examples:

* Service binding

  ```yaml
  apiVersion: services.cloud.sap.com/v1
  kind: ServiceBinding
  metadata:
    name: {SERVICE_BINDING}
  spec:
    serviceInstanceName: {SERVICE_INSTANCE_NAME}
  ```
* Secret <!--in line 40, should it be uri? in the created service bhinding secret, there was no uri, there was sburl; should I change it?-->

  ```yaml
  apiVersion: v1
  metadata:
    name: {SERVICE_BINDING}
  kind: Secret
  data:
    uri: {URI}
    client_id: {CLIENT_ID}
    client_secret: {CLIENT_SECRET}
    instance_guid: {SERVICE_INSTANCE_ID}
    instance_name: {SERVICE_INSTANCE_NAME}
    plan: {SERVICE_PLAN_NAME}               
    type: {SERVICE_OFFERING_NAME}  
  ```

## Credentials as a JSON Object

To show credentials that the service broker returns within the Secret resource as a JSON object, use the **secretKey** attribute in the service binding `spec`.
The value of the **secretKey** is the name of the key that stores the credentials. The credentials are represented in both formats: YAML or JSON.
See the following examples:

* Service binding

  ```yaml
  apiVersion: services.cloud.sap.com/v1
  kind: ServiceBinding
  metadata:
    name: {SERVICE_BINDING}
  spec:
    serviceInstanceName: {SERVICE_INSTANCE_NAME}
    secretKey: myCredentials
  ```
* Secret

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {SERVICE_BINDING}
  data:
      myCredentials:
        uri: {URI}
        client_id: {CLIENT_ID},
        client_secret: {CLIENT_SECRET}
      instance_guid: {SERVICE_INSTANCE_ID}
      instance_name: {SERVICE_BINDING}
      plan: {SERVICE_PLAN_NAME}
      type: {SERVICE_OFFERING_NAME}
  ```

## Credentials and Service Info as One JSON Object

To show both credentials returned from the service broker and additional **ServiceInstance** attributes as a JSON object, use the **secretRootKey** attribute in the service binding spec.

The **secretRootKey** value is the name of the key that stores both credentials and service instance info. The credentials are represented in both formats: YAML or JSON.
See the following examples:

* Service binding

  ```yaml
  apiVersion: services.cloud.sap.com/v1
  kind: ServiceBinding
  metadata:
    name: {SERVICE_BINDING}
  spec:
    serviceInstanceName: {SERVICE_INSTANCE_NAME}
    secretRootKey: myCredentialsAndInstance
  ```

* Secret

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {SERVICE_BINDING}
  data:
      myCredentialsAndInstance:
          uri: {URI}
          client_id: {CLIENT_ID}
          client_secret: {CLIENT_SECRET}
          instance_guid: {SERVICE_INSTANCE_ID}
          instance_name: {SERVICE_INSTANCE_NAME}
          plan: {SERVICE_PLAN_NAME}
          type: {SERVICE_OFFERING_NAME}
  ```

## Custom Formats 

For additional flexibility, you can model the Secret resources according to your needs. 
To generate a custom-formatted Secret, use the **secretTemplate** attribute in the ServiceBinding spec.
This attribute expects a Go template as its value. For more information, see [Go Templates](https://pkg.go.dev/text/template).<br>
Ensure the template is in the YAML format and has the structure of a Kubernetes Secret. 

In the provided Secret, you can customize the `metadata` and `data` sections with the following options:

- `metadata`: labels and annotations
- `data`: customize or utilize one of the available formatting options following the instructions in the [Overview](#overview) section.


> [!NOTE]  
> If you customize `data`, it takes precedence over the provided pre-defined formats.

The provided templates are executed on a map with the following available attributes:

| Reference         | Description                                |                                                                          
|-----------------|--------------------------------------------|
| **instance.instance_guid** |  The service instance ID.     |
| **instance.instance_name** |  The service instance name.   |                                                
| **instance.plan**   |  The name of the service plan used to create this service instance. |  
| **instance.type**   |  The name of the associated service offering. |  
| **credentials.attributes(var)**   |  The content of the credentials depends on a service. For more details, refer to the documentation of the service you're using. |  
| **instance.label**  | The service offering name.  |

The following examples demonstrate the ServiceBinding and generated Secret resources:

* Service binding with customized `metadata` and `data` sections:

    In this example, you specify both `metadata` and `data` in the `secretTemplate`:

    * Service binding

      ```yaml
      apiVersion: services.cloud.sap.com/v1
      kind: ServiceBinding
      metadata:
        name: {SERVICE_BINDING}
      spec:
        serviceInstanceName: {SERVICE_INSTANCE_NAME}
        secretTemplate: |
          apiVersion: v1
          kind: Secret
          metadata:
            labels:
              service_plan: {{ .instance.plan }}
            annotations:
              instance: {{ .instance.instance_name }}
          data:
            USERNAME: {{ .credentials.client_id }}
            PASSWORD: {{ .credentials.client_secret }}
      ```

    * Secret

      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        labels:
          service_plan: {SERVICE_PLAN_NAME}
        annotations:
          instance: {SERVICE_INSTANCE_NAME}
      data:
        USERNAME: {CLIENT_ID}
        PASSWORD: {CLIENT_SECRET}
      ```

* Example of a binding with a customized `metadata` section and applied pre-existing formatting option for `data` with credentials as a JSON object:

    In this example, you omit `data` from the `secretTemplate` and use the `secretKey` to format your `data` instead.

    * Service binding

      ```yaml
      apiVersion: services.cloud.sap.com/v1
      kind: ServiceBinding
      metadata:
        name: {SERVICE_BINDING}
      spec:
        serviceInstanceName: {SERVICE_INSTANCE_NAME}
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

    * Secret

      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        labels:
          service_plan: {SERVICE_PLAN_NAME}
        annotations:
          instance: {SERVICE_INSTANCE_NAME}
      data:
        myCredentials:
          uri: {URI}
          client_id: {CLIENT_ID}
          client_secret: {CLIENT_SECRET}
        instance_guid: {SERVICE_INSTANCE_ID}
        instance_name: {SERVICE_INSTANCE_NAME}
        plan: {SERVICE_PLAN_NAME}
        type: {SERVICE_OFFERING_NAME}
        ```
    