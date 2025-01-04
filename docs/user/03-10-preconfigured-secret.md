# Preconfigured Credentials and Access

When you create SAP BTP, Kyma runtime, all necessary resources for consuming SAP BTP services are created, and the basic cluster access is configured.

## Credentials

When you create a Kyma instance in the SAP BTP cockpit, the following events happen in your subaccount:

1. An SAP Service Manager service instance with the `service-operator-access` plan is created.
2. An SAP Service Manager service binding with access credentials for the SAP BTP Operator is created.
3. The credentials from the service binding are passed on to the Kyma service instance in the creation process.
4. The `sap-btp-manager` Secret is created and managed in the `kyma-system` namespace.
5. The SAP BTP Operator module is installed by default together with:

   * The `sap-btp-manager` Secret.
   * The `sap-btp-service-operator` Secret with the access credentials for the SAP BTP service operator. You can view the credentials in the `kyma-system` namespace.
   * The `sap-btp-operator-config` ConfigMap.

> [!TIP] <!--OS only-->
> In this scenario, the `sap-btp-service-operator` Secret is automatically generated when you create Kyma runtime. To create this Secret manually for a specific namespace, see [Create a Namespace-Based Secret](03-22-namespace-level-mapping.md#create-a-namespace-based-secret).

The `sap-btp-manager` Secret provides the following credentials:

* **clientid**
* **clientsecret**
* **cluster_id**
* **sm_url**
* **tokenurl**

> [!NOTE]
> If you modify or delete the `sap-btp-manager` Secret, it is modified back to its previous settings or regenerated within up to 24 hours.
> However, if the Secret is labeled with `kyma-project.io/skip-reconciliation: "true"`, the job skips the reconciliation for this Secret.

When you add the SAP BTP Operator module to your cluster, the `sap-btp-manager` Secret generates the SAP BTP service operator's resources as shown in the following diagram:
<!-- for the HP doc this sentence is different: The SAP BTP Operator module is added by default to your cluster and the `sap-btp-manager` (...) -->

![module_credentials](../assets/module_credentials.drawio.svg)

The cluster ID represents a Kyma service instance created in a particular subaccount and allows for its identification. You can view the cluster ID in the SAP BTP cockpit:
* In the `sap-btp-manager` Secret
* In the `sap-btp-service-operator` Secret
* In the `sap-btp-operator-config` ConfigMap

## Cluster Access

By default, SAP BTP Operator has cluster-wide permissions. You cannot reconfigure the predefined settings.

The following parameters manage cluster access:

| Parameter                     | Description                                                                                   |
|-------------------------------|-----------------------------------------------------------------------------------------------|
| **CLUSTER_ID**                | Generated when Kyma runtime is created.                                                       |
| **MANAGEMENT_NAMESPACE**      | Always set to `kyma-system`.                                                |
| **ALLOW_CLUSTER_ACCESS**      | You can use every namespace for your operations. The parameter is always set to `true`.<br>If you change it to `false`, the setting is automatically reverted. |
