# Create a Service Instance with a Namespace-Based Secret

To have service instances from one subaccount associated with one namespace, you must use a Secret dedicated to this namespace to create these service instances.

## Procedure

### Create a Namespace-Based Secret

1. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its service binding. Copy them from the BTP cockpit as a JSON.

2. Create the `creds.json` file in your working directory and save the credentials there.

3. In the same working directory, generate the Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **managed namespace sap-btp-service-operator secret**  as the second parameter.

    ```sh
    curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s operator {NAMESPACE_NAME}-sap-btp-service-operator
    ```

    The expected result is the file `btp-access-credentials-secret.yaml` created in your working directory:

    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
      name: {NAMESPACE_NAME}-sap-btp-service-operator
      namespace: kyma-system
    data:
      clientid: {CLIENT_ID}
      clientsecret: {CLIENT_SECRET}
      sm_url: {SM_URL}
      tokenurl: {AUTH_URL}
      tokenurlsuffix: "/oauth/token"
    ```
When you add the access credentials of the SAP Service Manager instance in your service instance, check the subaccount ID to which the instance belongs in the status **subaccountID** field.

### Create a Service Instance with a Managed Namespace Secret

Provide the needed parameters and create your service instance.

See an example of a ServiceInstance custom resource: <!-- why not placeholders??? REMOVE WHAT'S NOT NEEDED!!!!!-->

```yaml
kubectl create -f - <<EOF
apiVersion: services.cloud.sap.com/v1
kind: ServiceInstance
metadata:
  name: {SERVICE_INSTANCE_NAME}
  namespace: kyma-system
spec:
  serviceOfferingName: xsuaa {SERVICE_OFFERING_NAME}
  servicePlanName: application {SERVICE_PLAN_NAME}
  btpAccessCredentialsSecret: {YOUR_SECRET_NAME}
EOF
```

## Result

To verify that your service instance has been created successfully, run:

```bash
kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -o yaml
```

You see the status `Created` and the message `ServiceInstance provisioned successfully`.
You also see the Secret name in the **btpAccessCredentialsSecret** field of the `spec`.
In the status section, the **subaccountId** field must not be empty. <!-- move this sentence to line 36?? -->

## Related Information

[Working with Multiple Subaccounts](03-20-multitenancy.md)
