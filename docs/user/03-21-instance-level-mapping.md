# Instance-Level Mapping
<!--Instance-Level Association???--->
To have multiple service instances from different subaccounts associated with one namespace, you must use a custom Secret to create these service instances.

## Prerequisites

* A subaccount in the SAP BTP cockpit.
* kubectl configured for communicating with your Kyma instance.

## Context

To create a service instance with a custom Secret, you must use the **btpAccessCredentialsSecret** field in the `spec` of the service instance. In it, you pass the Secret from the `kyma-system` namespace to create your service instance. You can use different Secrets for different service instances.

## Procedure

### Create Your Custom Secret

1. In the SAP BTP cockpit, create an SAP Service Manager service instance with the `service-operator-access` plan.
2. Create a service binding to the SAP Service Manager service instance you have created.
3. Get the access credentials of the SAP Service Manager instance from its service binding. Copy them from the BTP cockpit as a JSON.
4. Create the `creds.json` file in your working directory and save the credentials there.
5. In the same working directory, generate the Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **your-secret-name**  as the second parameter:

    ```sh
    curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s operator {YOUR_SECRET_NAME}
    ```

    The expected result is the file `btp-access-credentials-secret.yaml` created in your working directory:

    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
      name: {YOUR_SECRET_NAME}
      namespace: kyma-system
    data:
      clientid: {CLIENT_ID}
      clientsecret: {CLIENT_SECRET}
      sm_url: {SM_URL}
      tokenurl: {AUTH_URL}
      tokenurlsuffix: "/oauth/token"
    ```

6. To create the Secret, run:

    ```
    kubectl create -f ./btp-access-credentials-secret.yaml
    ```

   You can see the status `Created`.

### Create a Service Instance with the Custom Secret

1. Create your service instance with:
   * the **btpAccessCredentialsSecret** field in the `spec` pointing to the custom Secret you have created
   * other parameters as needed<br>
    
    > [!WARNING] 
    > Once you set a Secret name in the service instance, you cannot change it in the future.

    See an example of a ServiceInstance custom resource:

    ```yaml
    kubectl create -f - <<EOF
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceInstance
    metadata:
      name: {SERVICE_INSTANCE_NAME}
      namespace: {NAMESPACE_NAME}
    spec:
      serviceOfferingName: {SERVICE_OFFERING_NAME}
      servicePlanName: {SERVICE_PLAN_NAME}
      btpAccessCredentialsSecret: {YOUR_SECRET_NAME}
    EOF
    ```

2. To verify that your service instance has been created successfully, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -o yaml
    ```

    You see the status `Created`.
    You also see your Secret name in the **btpAccessCredentialsSecret** field of the `spec`.

3.  To verify if you've correctly added the access credentials of the SAP Service Manager instance in your service instance, go to the CR `status` section, and make sure the subaccount ID to which the instance belongs is provided in the **subaccountID** field. The field must not be empty.

## Related Information

[Working with Multiple Subaccounts](03-20-multitenancy.md)<br>
[Namespace-Based Mapping](03-22-namespace-based-mapping.md)
