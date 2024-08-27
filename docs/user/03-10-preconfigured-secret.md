# Preconfigured Credentials and Access

## Credentials

On enabling SAP BTP, Kyma runtime, all necessary resources for consuming SAP BTP services are created, and cluster access is configured.
<!-- ~~The SAP BTP Operator module configures and installs the SAP BTP service operator, which allows you to immediately create Kyma service instances and service bindings.~~ -->

When you click on `Enable Kyma` in the SAP BTP cockpit,in your subaccount, the following events happen:
1. An SAP Service Manager service instance with the `service-operator-access` plan is created.
2. An SAP Service Manager service binding with access credentials for the SAP BTP Operator is created by CIS <!--Do we need this info? What does it mean to the users? And why not Commercialization Infrastructure Services?-- >.
3. The credentials from the service binding are passed on to the Kyma service instance in the creation process.
4. The `sap-btp-manager` Secret is created and managed in the `kyma-system` namespace.

When you create a Kyma cluster, the SAP BTP Operator module is installed by default together with:
* The `sap-btp-manager` Secret.
* The `sap-btp-service-operator` Secret with the access credentials for the SAP BTP service operator. You can view the credentials in the `kyma-system` namespace.
* The `sap-btp-operator-config` Config Map.

> [!NOTE]
> In this scenario, the `sap-btp-service-operator` Secret is automatically created when you enable Kyma runtime. If you want to create this Secret manually for a specific namespaace, follow the instructions in [Create SAP BTP Service Operator Secret](./tutorials/04-20-create-btp-service-operator-secret.md).

The `sap-btp-manager` Secret provides the following credentials.
* Client ID
* Client Secret
* Cluster ID
* Service Manager URL
* Token URL

When you add the SAP BTP Operator module to your cluster, the `sap-btp-manager` Secret generates the SAP BTP service operator's resources <!--credentials?--> as shown in the following diagram:

![module_credentials](../assets/module_credentials.drawio.svg)

> [!NOTE]
> If you modify or delete the `sap-btp-manager` Secret, it is modified back to its previous settings or regenerated within up to 24 hours.

The cluster ID represents a Kyma service instance created in a particular cluster and allows for its identification. You can view a cluster ID in the SAP BTP cockpit:
* In the list of instances as your Kyma service instance's scope
* In the `sap-btp-manager` Secret
* In the `sap-btp-operator-config` Config Map

## Cluster Access

By default, SAP BTP Operator has cluster-wide permissions. Currently, reconfiguring the predefined settings is not possible.

The following table lists the parameters managing cluster acccess:

| Parameter                     | Description                                                                                   |
|-------------------------------|-----------------------------------------------------------------------------------------------|
| **CLUSTER_ID**                | Generated when Kyma runtime is created.                                                       |
| **MANAGEMENT_NAMESPACE**      | Always set to `kyma-system`.                                                |
| **ALLOW_CLUSTER_ACCESS**      | You can use every namespace for your operations. The parameter is always set to `true`.<br>If you change it to `false`, the setting is automatically reverted. |
