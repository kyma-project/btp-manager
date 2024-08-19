# SAP BTP Operator Custom Resource

The `btpoperators.operator.kyma-project.io` CustomResourceDefinition (CRD) is a comprehensive specification that defines the structure and format used to manage the configuration and status of the SAP BTP Operator module within your Kyma environment.

To get the latest CRD in the YAML format, run the following command:

```shell
kubectl get crd btpoperators.operator.kyma-project.io -o yaml
```
You can only have one SAP BTP Operator (`BtpOperator`) CR. If there are multiple BtpOperator CRs in the cluster, the oldest one reconciles the module. An additional BtpOperator CR has the `Warning` state.

## Sample Custom Resource

The following <!-- SAP BTP Operator? BtpOperator?--> object defines a module:



## Custom Resource Parameters

The following table lists the parameters of the given resource with their descriptions:

| Parameter             | Type   | Description                                                                                                                                    |
|-----------------------|--------|------------------------------------------------------------------------------------------------------------------------------------------------|
|