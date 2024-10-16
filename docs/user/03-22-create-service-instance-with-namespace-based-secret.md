# Create a Service Instance with a Namespace-Based Secret

To have service instances from one subaccount associated with one namespace, you must use a Secret dedicated to this namespace to create these service instances.

## Prerequisites

* A subaccount in the SAP BTP cockpit

## Procedure

### Create a Namespace-Based Secret

1. [Create an SAP Service Manager service instance](03-30-management-of-service-instances-and-bindings.md#create-a-service-instance) with the `service-operator-access` plan.
2. [Create a service binding](03-30-management-of-service-instances-and-bindings.md#create-a-service-binding) to the SAP Service Manager service instance you have created.
3. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its service binding. Copy them from the SAP BTP cockpit as a JSON.
4. Create the `creds.json` file in your working directory and save the credentials there.
5. In the same working directory, generate the Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **managed namespace sap-btp-service-operator secret**  as the second parameter.

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
6. To verify if you've correctly added the access credentials of the SAP Service Manager instance in your service instance, go to the CR `status` section, and make sure the subaccount ID to which the instance belongs is provided in the **subaccountID** field. The field must not be empty.

### Create a Service Instance with a Managed Namespace Secret

1. Provide the needed parameters and create your service instance.

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
    EOF
    ```

2. To verify that your service instance has been created successfully, run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -o yaml
    ```

## Result

You see the status `Created` and the message confirming that your servicde instance was created successfully.

## Related Information

[Working with Multiple Subaccounts](03-20-multitenancy.md)<br>
[Create a Service Instance with a Custom Secret](03-21-create-service-instance-with-custom-secret.md)
