# Can't Delete Leftover Service Instances and Bindings

## Symptom

After your Kyma cluster has become unavailable, some orphaned service instances and service bindings are still present in the SAP Business Technology Platform (BTP).

## Cause

You cannot access your Kyma cluster where the service instances and bindings were created. Without access to the cluster, you cannot use standard deletion methods.

## Solution

> [!Warning]
> Before using this cleanup option, ensure that the cluster cannot be accessed. Otherwise, the operation can cause discrepancies between what remains in SAP BTP and what's running in your Kyma cluster, leading to orphaned resources.

1. To access the Service Manager API, perform the following actions:
   
   1. In the SAP BTP cockpit, create an SAP Service Manager service instance with the `subaccount-admin` plan and its service binding. The `service-operator-access` plan does not have the necessary access level for performing the cleanup operation. See [Create a SAP Service Manager Instance and Binding](https://help.sap.com/docs/service-manager/sap-service-manager/create-sap-service-manager-instance-and-binding).
   2. To get an access token, follow the instructions in [Retrieve an OAuth 2.0 Access Token](https://help.sap.com/docs/service-manager/sap-service-manager/retrieve-oauth2-access-token).

2. To prepare the DELETE request, identify the following parameters:
     - **platformID** - the ID of the platform (the ID of the Service Manager instance with the `service-operator-access` plan).
     - **clusterID** - the ID of your cluster. Retrieve it directly from the service binding. Alternatively, use a GET service instance or binding API call, or the BTP CLI command to extract the ID from the response.
 
3. Send the request:
   
   ```bash
   curl -X DELETE '{sm_url}v1/platforms/{platformID}/clusters/{clusterID}' \
   -H 'Authorization: Bearer {ACCESS_TOKEN}'
   ```

4. Monitor the response. You can get one of the following options:
   
    - `202 Accepted` - the request is accepted for processing.
    - `404 Resource Not Found` - platform or cluster not found.
    - `429 Too Many Requests` - the rate limit exceeded.

5. For the operation status, review the following headers:

   - **Location** - the path provided to monitor the status of the operation. For more information about operations, see [Service Manager operation API](https://api.sap.com/api/APIServiceManager/resource/getSingleOperation).
   - **Retry_After** - indicates the time in seconds after which you can retry the request after hitting rate limits.

## Related Information

- [Using the SAP BTP Command Line Interface (btp CLI)](https://help.sap.com/docs/service-manager/sap-service-manager/working-with-sap-service-manager-resources-by-using-sap-btp-command-line-interface-btp-cli-feature-set-b?version=Cloud)
