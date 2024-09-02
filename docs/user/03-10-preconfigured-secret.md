# Preconfigured Credentials and Access

## Credentials

On enabling SAP BTP, Kyma runtime, all necessary resources for consuming SAP BTP services are created, and the basic cluster access is configured.

When you click on `Enable Kyma` in the SAP BTP cockpit, the following events happen in your subaccount:
1. An SAP Service Manager service instance with the `service-operator-access` plan is created.
2. An SAP Service Manager service binding with access credentials for the SAP BTP Operator is created.
3. The credentials from the service binding are passed on to the Kyma service instance in the creation process.
4. The `sap-btp-manager` Secret is created and managed in the `kyma-system` namespace.

When you create a Kyma cluster, the SAP BTP Operator module is installed by default together with:
* The `sap-btp-manager` Secret.
* The `sap-btp-service-operator` Secret with the access credentials for the SAP BTP service operator. You can view the credentials.
* The `sap-btp-operator-config` Config Map.

> [!TIP]
> In this scenario, the `sap-btp-service-operator` Secret is automatically generated when you enable Kyma runtime. If you want to create this Secret manually for a specific namespace, follow the instructions in [Create an SAP BTP Service Operator Secret](./tutorials/04-20-create-btp-service-operator-secret.md).

The `sap-btp-manager` Secret provides the following credentials:
* **clientid**
* **clientsecret**
* **cluster_id**
* **sm_url**
* **tokenurl**

When you add the SAP BTP Operator module to your cluster, the `sap-btp-manager` Secret generates the SAP BTP service operator's resources as shown in the following diagram:

![module_credentials](../assets/module_credentials.drawio.svg)

> [!NOTE]
> If you modify or delete the `sap-btp-manager` Secret, it is modified back to its previous settings or regenerated within up to 24 hours.

The cluster ID represents a Kyma service instance created in a particular subacoount and allows for its identification. You can view a cluster ID in the SAP BTP cockpit:
* In the list of instances as your Kyma service instance's scope
* In the `sap-btp-manager` Secret
* In the `sap-btp-operator-config` Config Map

## Cluster Access

By default, SAP BTP Operator has cluster-wide permissions. Currently, reconfiguring the predefined settings is not possible.

The following table lists the parameters managing cluster access:

| Parameter                     | Description                                                                                   |
|-------------------------------|-----------------------------------------------------------------------------------------------|
| **CLUSTER_ID**                | Generated when Kyma runtime is created.                                                       |
| **MANAGEMENT_NAMESPACE**      | Always set to `kyma-system`.                                                |
| **ALLOW_CLUSTER_ACCESS**      | You can use every namespace for your operations. The parameter is always set to `true`.<br>If you change it to `false`, the setting is automatically reverted. |
