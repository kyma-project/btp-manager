# Preconfigured <!--what secret?--> Secret

The SAP BTP Operator module configures and installs the SAP BTP service operator, which allows you to immediately create Kyma service instances and service bindings.

When you click on `Enable Kyma` in the SAP BTP cockpit,in your subaccount, you trigger the creation of:<!-- do we need to mention "a call to the environment registry service which creates:", isn't it too much detail? -->
<!--what is the "environment registry service"? Is this: https://help.sap.com/doc/saphelp_nw74/7.4.16/en-us/47/d391d7b8fc3c83e10000000a42189c/frameset.htm??? -->
1. an SAP Service Manager service instance with the `service-operator-access` plan
2. an SAP Service Manager service binding with access credentials for the SAP BTP Operator including the `sap-btp-manager` Secret automatically created in the `kyma-system` namespace.

The SAP BTP Operator module is added by default when you create a Kyma cluster. The module performs the following actions:
* injects the `sap-btp-manager` Secret into your cluster
* installs the `sap-btp-service-operator` Secret with the access credentials for the SAP BTP service operator. You can view the credentials in the `kyma-system` namespace. <!-- is this sentence ok?-->
* installs the `sap-btp-operator-config` Config Map

The `sap-btp-manager` Secret provides the following credentials:
* Client ID
* Client Secret
* Cluster ID
* Service Manager <!--??--> URL
* Token URL

> [!NOTE]
> If you modify or delete the `sap-btp-manager` Secret, it is modified back to its previous version or regenerated within up to 24 hours.

The cluster ID represents a Kyma service instance created in a particular cluster and allows for its identification. You can view a cluster ID in the SAP BTP cockpit:
* in the list of instances as your Kyma service instance's scope
* in the `sap-btp-manager` Secret
* in the `sap-btp-operator-config` Config Map
