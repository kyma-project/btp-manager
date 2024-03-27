# Sap BTP Service Operator Examples

This document describes notable features shipped with new SAP BTP service operator upgrades discovered during the application of latest versions.

## Secret Templates

Version 0.6.1 introduced a Secret templates feature that allows for modification of Secret's content that is normally generated from a ServiceBinding instance created for any service. You can now specify a Go template as a value of the **secretTemplate** field inside the binding's spec section. Inside the template, you can refer (by using `{{}}` syntax) to credentials stored inside of a ServiceBinding (with `{{credentials.<key>}}`) or information of a given instance (with `{{instance.<key>}}`). The parameters that can be used with the `instance` key are limited to the values set in the [getInstanceInfo](https://github.com/SAP/sap-btp-service-operator/blob/8c0a3d7d7ca54e44143c0e0b7d1e1ef206b362ab/controllers/servicebinding_controller.go#L819) method. Here is an example of a ServiceBinding with the **secretTemplate** field:

```
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  labels:
    app.kubernetes.io/name: nice-script
  name: nice-script
  namespace: default
spec:
  externalName: nice-script
  secretName: nice-script
  secretTemplate: |
    apiVersion: v1
    kind: Secret
    metadata:
      labels:
        instance_plan: {{ .instance.plan }}
      annotations:
        instance_name: {{ .instance.instance_name }}
    data:
      foo: {{ .instance.type }}
      bar: {{ .credentials.url }}
  serviceInstanceName: dimpled-editor
```
