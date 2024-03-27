# Examples

This document describes notable features shipped with new SAP BTP service operator upgrades discovered during the application of latest versions.

## Secret Templates

Version 0.6.1 introduced secret templates feature that allows to modify contents of the secret that is normally generated from `ServiceBinding` instance created for any service. The customer can now specify a go template as a value of `secretTemplate` field inside of binding's spec section. Inside of the template you can refer (by using `{{}}` syntax) to credentials stored inside of a service binding (with `{{credentials.<key>}}`) or information of given instance (with `{{instance.<key>}}`). Parameter that can be used with `instance` key are limited to the values set in [getInstanceInfo](https://github.com/SAP/sap-btp-service-operator/blob/8c0a3d7d7ca54e44143c0e0b7d1e1ef206b362ab/controllers/servicebinding_controller.go#L819) method. Below is an example of a service binding with spectTemplate value field:

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
