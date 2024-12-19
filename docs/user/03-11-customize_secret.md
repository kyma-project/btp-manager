# Customize Credentials and Access

To use your own credentials as the default and manage your cluster on your own, customize the `sap-btp-manager` Secret.

## Context

When you create SAP BTP, Kyma runtime, the `sap-btp-manager` Secret is automatically created as the default Secret managing the SAP BTP service operator's resources. 
Any changes to the Secret are reverted, and the previous settings are restored within up to 24 hours.
See [Preconfigured Credentials and Access](03-10-preconfigured-secret.md).

To stop the reconciliation of the `sap-btp-manager` Secret, you must customize it. With the customized Secret, you can perform the following actions:

* Manage your Kyma runtime
* Migrate the instances you created outside of the Kyma environment to your Kyma cluster

## Procedure

To customize the `sap-btp-manager` Secret, perform the following steps:

* Label the Secret with `kyma-project.io/skip-reconciliation: 'true'`.
* Provide the **cluster_id** of <!-- or Provide the credentials of???--> the Kyma instance you want to migrate/recreate.
* Optionally, add the `management_namespace` parameter <!--to the `data` object--> and provide the name of your custom management namespace.

## Result

When you configure your custom `sap-btp-manager` Secret, it generates the SAP BTP service operator's resources as shown in the following diagram:

![Customized module credentials](../assets/module_credentials_customized.drawio.svg)

The reconciliation of the Secret stops and your changes are not reverted.

> [!WARNING]
> If you delete the customized `sap-btp-manager` Secret, the reconciliation starts again, and the default `sap-btp-manager` Secret is recreated for your Kyma instance within 24 hours.

## Next Steps

Now, you can manage your Kyma cluster. For example, you can assign multiple namespaces to the Secret or delete your Kyma cluster regardless of the existing service instances and bindings associated with your `sap-btp-manager` Secret.
You can also recreate your instances in the Kyma environment. For example, you can migrate your instance from Gardener to the Kyma environment.