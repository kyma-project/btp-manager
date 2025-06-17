# Migrate Service Instances and Service Bindings from custom SAP BTP service operator
The following guide describes the migration process of Service Instances and Service Bindings from the custom SAP BTP service operator to the Kyma cluster.

## Prepare migration data from custom SAP BTP service operator
> [!NOTE] 
> Make sure kubectl is connected to the cluster with custom SAP BTP service operator by setting the **KUBECONFIG** environment variable or setting cluster context with `kubectl config use-context` command.

1. Find **sap-btp-service-operator** secret in the unmanaged cluster by running `kubectl get secrets -A` command.
2. Save the secret name and namespace in **SAP_BTP_OPERATOR_SECRET_NAME**, **SAP_BTP_OPERATOR_SECRET_NAMESPACE** environment variables.
3. Save SAP BTP service operator credentials in environment variables:
```
CLIENT_ID=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.clientid})
CLIENT_SECRET=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.clientsecret})
SM_URL=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.sm_url})
TOKEN_URL=$(kubectl get secret -n $SAP_BTP_OPERATOR_SECRET_NAMESPACE $SAP_BTP_OPERATOR_SECRET_NAME -o jsonpath={.data.tokenurl})
```
4. Find **sap-btp-operator-config** configmap in the unmanaged cluster by running `kubectl get configmaps -A` command.
5. Save the configmap name and namespace in **SAP_BTP_OPERATOR_CONFIGMAP_NAME**, **SAP_BTP_OPERATOR_CONFIGMAP_NAMESPACE** environment variables.
6. Save cluster ID in a environment variable:
```
CLUSTER_ID=$(kubectl get configmap -n $SAP_BTP_OPERATOR_CONFIGMAP_NAMESPACE $SAP_BTP_OPERATOR_CONFIGMAP_NAME -o jsonpath={.data.CLUSTER_ID} | base64)
```
7. List all Service Instances with `kubectl get serviceinstances -A` command. Take a note of namespaces which must be present in the Kyma cluster.
8. Save each Service Instance you want to migrate as a manifest in a JSON file. You can run the following command:
```
kubectl get serviceinstance -n <SERVICE_INSTANCE_NAMESPACE> <SERVICE_INSTANCE_NAME> -o json \
| jq 'del(.metadata.annotations, .metadata.creationTimestamp, .metadata.finalizers, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .status)' \
> <SERVICE_INSTANCE_NAME>-si.json
```
where **SERVICE_INSTANCE_NAME** and **SERVICE_INSTANCE_NAMESPACE** are placeholders for the actual Service Instance name and namespace.

9. List all Service Bindings with `kubectl get servicebindings -A` command. Take a note of namespaces which must be present in the Kyma cluster.
10. Save each Service Binding you want to migrate as a manifest in a JSON file. You can run the following command:
```
kubectl get servicebinding -n <SERVICE_BINDING_NAMESPACE> <SERVICE_BINDING_NAME> -o json \
| jq 'del(.metadata.annotations, .metadata.creationTimestamp, .metadata.finalizers, .metadata.generation, .metadata.ownerReferences, .metadata.resourceVersion, .metadata.uid, .status)' \
> <SERVICE_BINDING_NAME>-sb.json
```
where **SERVICE_BINDING_NAME** and **SERVICE_BINDING_NAMESPACE** are placeholders for the actual Service Binding name and namespace.

## Migrate resources to Kyma cluster
> [!NOTE] 
> Make sure kubectl is connected to the Kyma cluster by setting the **KUBECONFIG** environment variable or setting cluster context with `kubectl config use-context` command.

The following steps concern the `sap-btp-manager` secret customization. Make sure the secret has `kyma-project.io/skip-reconciliation: 'true'` label. See [Customize the Default Credentials and Access](03-11-customize_secret.md) for details.

1. Find **sap-btp-manager** secret in the Kyma cluster by running `kubectl get secrets -A` command.
2. Save the secret name and namespace in **BTP_MANAGER_SECRET_NAME**, **BTP_MANAGER_SECRET_NAMESPACE** environment variables.
3. Create all required namespaces for Service Instances and Service Bindings.
4. Patch the BTP Manager secret to override SAP BTP service operator credentials and cluster ID:
```
kubectl patch secret -n ${BTP_MANAGER_SECRET_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${CLIENT_ID}\",\"clientsecret\":\"${CLIENT_SECRET}\",\"sm_url\":\"${SM_URL}\",\"tokenurl\":\"${TOKEN_URL}\",\"cluster_id\":\"${CLUSTER_ID}\"}}"
```
5. Create Service Instances and Service Bindings from saved manifests in JSON format by using `kubectl apply -f <JSON_MANIFEST>` command. **JSON_MANIFEST** is a placeholder for the actual Service Instance or Service Binding JSON manifest.

## Required cleanup 
> [!NOTE] 
> Make sure kubectl is connected to the cluster with custom SAP BTP service operator by setting the **KUBECONFIG** environment variable or setting cluster context with `kubectl config use-context` command.

The following steps are required to limit Service Instances and Service Bindings management to SAP BTP service operator in the Kyma cluster.

1. Find **sap-btp-operator-controller-manager** deployment in the unmanaged cluster by running `kubectl get deployments -A` command.
2. Save the deployment name and namespace in **SAP_BTP_OPERATOR_DEPLOYMENT_NAME**, **SAP_BTP_OPERATOR_DEPLOYMENT_NAMESPACE** environment variables.
3. Scale the deployment to 0 replicas by running the following command: 
```
kubectl scale deployment -n $SAP_BTP_OPERATOR_DEPLOYMENT_NAMESPACE $SAP_BTP_OPERATOR_DEPLOYMENT_NAME --replicas=0
```
4. Delete SAP BTP service operator webhooks by running the following command:
```
kubectl delete mutatingwebhookconfigurations sap-btp-operator-mutating-webhook-configuration && kubectl delete validatingwebhookconfigurations sap-btp-operator-validating-webhook-configuration
```
5. Delete finalizers from each migrated Service Instance and Service Binding.
6. Delete migrated Service Bindings.
7. Delete migrated Service Instances.