# Customize the Default Credentials and Access

You can customize the `sap-btp-manager` Secret and manage your own default configuration of the SAP BTP Operator module.

## Context

When you create SAP BTP, Kyma runtime, the `sap-btp-manager` Secret is automatically created as the default Secret managing the SAP BTP Operator's resources. 
Because of Kyma's automatic reconciliation, any changes to the Secret are reverted, and the previous settings are restored within up to 24 hours.
See [Preconfigured Credentials and Access](03-10-preconfigured-secret.md#credentials).

To customize the `sap-btp-manager` Secret and prevent your changes from being reverted, you must stop the Secret's reconciliation.
With the customized Secret, you can perform the following actions:

* Manage your own default configuration of SAP BTP Operator
* Migrate the service instances you created outside of the Kyma environment to your Kyma cluster

## Procedure

To customize the `sap-btp-manager` Secret, perform the following steps:

* Label the Secret with `kyma-project.io/skip-reconciliation: 'true'`.
* Provide the following credentials from your SAP Service Manager instance: **clientid**, **clientsecret**, **sm_url**, and **tokenurl**.
* Optionally, provide your **cluster_id**. Otherwise, it is generated automatically.
* Optionally, add the `management_namespace` parameter and provide the name of your custom management namespace.

## Result

Your customized `sap-btp-manager` Secret is now the default Secret of the SAP BTP Operator module. It generates the SAP BTP service operator's resources, as shown in the following diagram:

![Customized module credentials](../assets/module_credentials_customized.drawio.svg)

The reconciliation of the Secret stops and your changes are not reverted.

> [!WARNING]
> If you delete the customized `sap-btp-manager` Secret, the reconciliation starts again, and the preconfigured default `sap-btp-manager` Secret is recreated for your Kyma instance within 24 hours. See [Preconfigured Credentials and Access](./03-10-preconfigured-secret.md#credentials).

> [!NOTE]
> If you created all service instances in your Kyma cluster from the customized `sap-btp-manager` Secret, you can delete the cluster even if those instances still exist.
> The undeleted service instances do not block the deletion of the cluster.
