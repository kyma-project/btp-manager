# Customize the Default Credentials and Access

You can customize the `sap-btp-manager` Secret and manage your own SAP BTP Operator's default configuration.

## Context

When you create SAP BTP, Kyma runtime, the `sap-btp-manager` Secret is automatically created as the default Secret managing the SAP BTP Operator's resources. 
Any changes to the Secret are reverted, and the previous settings are restored within up to 24 hours.
See [Preconfigured Credentials and Access](03-10-preconfigured-secret.md#credentials).

To customize the `sap-btp-manager` Secret and prevent your changes from being reverted, you must stop the Secret's reconciliation.
With the customized Secret, you can perform the following actions:

* Manage your own SAP BTP Operator's default configuration
* Migrate the instances you created outside of the Kyma environment to your Kyma cluster

## Procedure

To customize the `sap-btp-manager` Secret, perform the following steps:

* Label the Secret with `kyma-project.io/skip-reconciliation: 'true'`.
* Provide the credentials of the Kyma instance you want to migrate or recreate.
* Optionally, add the `management_namespace` parameter and provide the name of your custom management namespace.

## Result

Your customized `sap-btp-manager` Secret is now the default SAP BTP Operator's Secret. It generates the SAP BTP service operator's resources, as shown in the following diagram:

![Customized module credentials](../assets/module_credentials_customized.drawio.svg)

The reconciliation of the Secret stops and your changes are not reverted.

> [!WARNING]
> If you delete the customized `sap-btp-manager` Secret, the reconciliation starts again, and the preconfigured default `sap-btp-manager` Secret is recreated for your Kyma instance within 24 hours. See [Preconfigured Credentials and Access](./03-10-preconfigured-secret.md#credentials).

## Next Steps

With the customized default `sap-btp-manager` Secret, you can perform the following actions:

* Migrate your instances created, for example, in Gardener, to the Kyma environment.
* Connect a namespace to a specific subaccount without creating a namespace-based Secret because the `sap-btp-service-operator` Secret already includes your custom credentials. See [Namespace-Level Mapping](03-22-namespace-level-mapping.md).

> [!NOTE]
> If you have created a new non-default SAP Service Manager service instance and used its credentials to create your customized `sap-btp-manager` Secret, you can delete your Kyma cluster associated with this Secret regardless of the existing service instances and bindings created in this cluster.
> The undeleted service instances or bindings do not block the deletion of the cluster.
