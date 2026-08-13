# SAP BTP Operator Not Working After Deleting the Service Manager Binding

## Symptom

You deleted the SAP Service Manager service binding in the SAP BTP cockpit and the SAP BTP Operator module keeps failing. Attempts to create new service instances or bindings return errors.

## Cause

When your Kyma runtime was provisioned, an SAP Service Manager service instance with the `service-operator-access` plan and a corresponding service binding were created automatically. The credentials from that binding are used to populate the `sap-btp-manager` Secret in your cluster, which the SAP BTP Operator module relies on to function.

When you delete the binding, the Kyma infrastructure still holds the old credentials and keeps refreshing the `sap-btp-manager` Secret with them. If you re-create the binding, the new binding has different credentials, but the Secret is still not updated. The SAP BTP Operator module can't authenticate, so the entire SAP BTP module remains non-functional until the operations team manually updates the credentials on your behalf.

Alternatively, if the `sap-btp-manager` Secret itself was deleted from the cluster, Kyma's automatic reconciliation restores it within 24 hours — no support ticket is needed. If you can't wait, you can restore the Secret immediately by following [Customize the Default Credentials and Access](../03-11-customize_secret.md) using the credentials from the existing binding.

Each Kyma runtime has a dedicated Service Manager instance in a one-to-one relationship. If you have multiple Kyma runtimes, you have multiple Service Manager instances, each with its own binding. The Service Manager instance name in the SAP BTP cockpit matches the Kyma instance ID of the corresponding Kyma runtime. You need this mapping to identify the correct instance when raising a support ticket.

## Solution

1. In the SAP BTP cockpit, open your Kyma runtime and note its instance ID.

2. In the list of SAP Service Manager, find the service instance whose name matches your Kyma instance ID. This is the dedicated Service Manager instance for your Kyma runtime.

3. In that Service Manager instance, check whether a binding exists.
   - If no binding exists, re-create it, then proceed to the next step.
   - If a binding already exists (whether original or re-created), proceed to the next step.

4. Check the binding's creation timestamp and compare it with the Service Manager instance's creation timestamp. If the binding was created later, it was deleted and re-created — this confirms you're in the scenario described by this guide.

   > [!NOTE]
   > If the timestamps match, the binding has not been re-created and this guide does not apply. The issue may instead be caused by the `sap-btp-manager` Secret being deleted from the cluster. See the [Cause](#cause) section for guidance.

5. Download the binding credentials as a JSON file.

6. Create a support ticket and attach the credentials JSON file. Include the following information in the ticket:
   - Your subaccount ID
   - Your Kyma runtime instance ID
   - The Service Manager instance name (which equals your Kyma instance ID)

7. Wait for the operations team to confirm that the credentials have been updated and the `sap-btp-manager` Secret on your cluster has been restored. Then verify that the SAP BTP Operator module is functioning again by checking that your service instances and bindings are no longer reporting errors.
