# SAP BTP Operator Not Working After Deleting the Service Binding

## Symptom

You deleted the SAP Service Manager service binding in the SAP BTP cockpit and re-created it, but the SAP BTP Operator on your Kyma cluster keeps failing. Attempts to create new service instances or bindings return errors.

## Cause

When your Kyma runtime was provisioned, an SAP Service Manager service instance with the `service-operator-access` plan and a corresponding service binding were created automatically. The credentials from that binding are used to populate the `sap-btp-manager` Secret in your cluster, which the SAP BTP Operator relies on to function.

When you delete the binding and re-create it, the new binding has different credentials. However, the Kyma infrastructure still holds the old ones and keeps refreshing the `sap-btp-manager` Secret with the now-invalid credentials. The SAP BTP Operator can't authenticate, so the entire SAP BTP module remains non-functional until the operations team manually updates the credentials on your behalf.

> [!NOTE]
> The service instance itself can't be deleted from the SAP BTP cockpit. Only the binding can be deleted. Re-creating the binding is safe and doesn't affect your existing service instances.

## Solution

1. In the SAP BTP cockpit, navigate to the service binding you re-created and download the credentials as a JSON file. Make sure you're using the binding for the `service-operator-access` instance that was created automatically for your Kyma runtime. Using credentials from a different instance won't resolve the issue.

2. Create a support ticket and attach the credentials JSON file. Include the following information in the ticket:
   - Your subaccount ID
   - Your Kyma runtime instance ID

3. Wait for the operations team to confirm that the credentials have been updated and the `sap-btp-manager` Secret on your cluster has been restored. Then verify that the SAP BTP Operator is functioning again by checking that your service instances and bindings are no longer reporting errors.

## Related Information

[Customize the Default Credentials and Access](03-11-customize_secret.md)
