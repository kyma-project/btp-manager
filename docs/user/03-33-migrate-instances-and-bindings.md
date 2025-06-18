# Migrate Service Instances and Service Bindings from a Custom SAP BTP Service Operator
Learn how to migrate service instances and service bindings from a custom SAP BTP service operator to a Kyma cluster.

## Prepare Migration Data from Your Custom SAP BTP Service Operator
> [!NOTE] 
> Ensure that kubectl is connected to the cluster with your custom SAP BTP service operator by setting either the **KUBECONFIG** environment variable or the cluster context with the `kubectl config use-context` command.

1. In the cluster whose resources you want to migrate, find the `sap-btp-service-operator` Secret by running the `kubectl get secrets -A` command.
2. Save the Secret name and its namespace in the **SAP_BTP_OPERATOR_SECRET_NAME** and **SAP_BTP_OPERATOR_SECRET_NAMESPACE** environment variables.
3. Save the SAP BTP service operator credentials in the following environment variables:

    ```
    CLIENT_ID=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.clientid})
    CLIENT_SECRET=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.clientsecret})
    SM_URL=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.sm_url})
    TOKEN_URL=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.tokenurl})
    ```

4. To find the `sap-btp-operator-config` ConfigMap in the cluster, run the `kubectl get configmaps -A` command.
5. Save the ConfigMap name and its namespace in the **SAP_BTP_OPERATOR_CONFIGMAP_NAME** and **SAP_BTP_OPERATOR_CONFIGMAP_NAMESPACE** environment variables.
6. Save the cluster ID in the environment variable:

    ```
    CLUSTER_ID=$(kubectl get configmap -n $SAP_BTP_OPERATOR_CONFIGMAP_NAMESPACE $SAP_BTP_OPERATOR_CONFIGMAP_NAME -o jsonpath={.data.CLUSTER_ID} | base64)
    ```

7. List all service instances with the `kubectl get serviceinstances -A` command. Take note of the namespaces that must be present in the Kyma cluster.
8. Save each service instance you want to migrate as a manifest in a JSON file. To do that, run:

    ```
    kubectl get serviceinstance -n <SERVICE_INSTANCE_NAMESPACE> <SERVICE_INSTANCE_NAME> -o json \
    | jq 'del(.metadata.annotations, .metadata.creationTimestamp, .metadata.finalizers, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .status)' \
    > <SERVICE_INSTANCE_NAME>-si.json
    ```

    where **SERVICE_INSTANCE_NAME** and **SERVICE_INSTANCE_NAMESPACE** are placeholders for the actual service instance name and its namespace.

9. List all service bindings with the `kubectl get servicebindings -A` command. Take note of namespaces that must be present in the Kyma cluster.
10. Save each service binding you want to migrate as a manifest in a JSON file. To do that, run:

    ```
    kubectl get servicebinding -n <SERVICE_BINDING_NAMESPACE> <SERVICE_BINDING_NAME> -o json \
    | jq 'del(.metadata.annotations, .metadata.creationTimestamp, .metadata.finalizers, .metadata.generation, .metadata.ownerReferences, .metadata.resourceVersion, .metadata.uid, .status)' \
    > <SERVICE_BINDING_NAME>-sb.json
    ```

    where **SERVICE_BINDING_NAME** and **SERVICE_BINDING_NAMESPACE** are placeholders for the actual service binding name and its namespace.

## Migrate Resources to a Kyma Cluster
> [!NOTE] 
> Ensure that kubectl is connected to your Kyma cluster by setting either the **KUBECONFIG** environment variable or the cluster context with the `kubectl config use-context` command.

To migrate your resources to a Kyma cluster, you must first customize the `sap-btp-manager` Secret. To prevent automatic reversion of your custom changes, add the `kyma-project.io/skip-reconciliation: 'true'` label to the Secret and perform the following steps:

1. To find the `sap-btp-manager` Secret in the Kyma cluster, run the `kubectl get secrets -A` command.
2. Save the Secret name and its namespace in the **BTP_MANAGER_SECRET_NAME** and **BTP_MANAGER_SECRET_NAMESPACE** environment variables.
3. Create all required namespaces for the service instances and service bindings you are migrating.
4. To override SAP BTP service operator credentials and cluster ID, patch the `sap-btp-manager` Secret:

    ```
    kubectl patch secret -n ${BTP_MANAGER_SECRET_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${CLIENT_ID}\",\"clientsecret\":\"${CLIENT_SECRET}\",\"sm_url\":\"${SM_URL}\",\"tokenurl\":\"${TOKEN_URL}\",\"cluster_id\":\"${CLUSTER_ID}\"}}"
    ```

5. To create service instances and service bindings from the saved manifests in JSON format, use the `kubectl apply -f <JSON_MANIFEST>` command. **JSON_MANIFEST** is a placeholder for the actual service instance or service binding JSON manifest.

## Required Cleanup 
> [!NOTE] 
> Ensure that kubectl is connected to the cluster with your custom SAP BTP service operator by setting either the **KUBECONFIG** environment variable or the cluster context with the `kubectl config use-context` command.

To limit service instances and service bindings management to the SAP BTP service operator in the Kyma cluster, perform the following steps:

1. To find the `sap-btp-operator-controller-manager` deployment in the cluster with your custom SAP BTP service operator, run the `kubectl get deployments -A` command.
2. Save the deployment name and its namespace in the **SAP_BTP_OPERATOR_DEPLOYMENT_NAME** and **SAP_BTP_OPERATOR_DEPLOYMENT_NAMESPACE** environment variables.
3. To scale the deployment to 0 replicas, run: 

    ```
    kubectl scale deployment -n $SAP_BTP_OPERATOR_DEPLOYMENT_NAMESPACE $SAP_BTP_OPERATOR_DEPLOYMENT_NAME --replicas=0
    ```

4. To delete SAP BTP service operator webhooks, run:

    ```
    kubectl delete mutatingwebhookconfigurations sap-btp-operator-mutating-webhook-configuration && kubectl delete validatingwebhookconfigurations sap-btp-operator-validating-webhook-configuration
    ```

5. Delete finalizers from each migrated service instance and service binding.
6. Delete migrated service bindings.
7. Delete migrated service instances.

## Read More
[Customize the Default Credentials and Access](03-11-customize_secret.md)