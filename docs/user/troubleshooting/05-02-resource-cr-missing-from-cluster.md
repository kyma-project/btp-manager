# Resource CR Missing from the Cluster

## Symptom

Your service instance or binding is visible in SAP BTP, but its corresponding custom resource (CR) is missing in your Kyma cluster.

## Cause

The CR representing an SAP BTP service instance or service binding is absent from the Kyma cluster. This happens if the CR was accidentally deleted or if there are issues preventing its synchronization or visibility in the cluster. While the resource still exists, the connection to your Kyma cluster is broken due to the absence of the CR.

## Solution

To address this issue, you can manually recreate the service instance CR or service binding CR using details from SAP BTP to restore the connection with the existing BTP resource. For successful reconnection, ensure that the new CR matches the name, resides in the same namespace, and is linked to the same cluster ID as the original CR. Since the SAP BTP resource maintains its configuration, matching these attributes allows the new CR to reconnect to the existing BTP resource without provisioning a new one.

To restore the CR, follow these steps:

1. Retrieve the following CR details from the service instance or binding:

   - The CR name
   - The name of the namespace where the CR should reside

2. Create your YAML manifest for the CR, including the exact name and namespace you retrieved from the SAP BTP service instance or binding.
3. To recreate the CR in your Kyma cluster, run:
   
   ```bash
   kubectl apply -f {YAML_MANIFEST}
   ```

4. To verify recreation of the CR in your Kyma cluster, replace the placeholders and run:

    ```bash
    kubectl get serviceinstances.services.cloud.sap.com {SERVICE_INSTANCE_NAME} -n {NAMESPACE}
    ```

    Or

    ```bash
    kubectl get servicebindings.services.cloud.sap.com {BINDING_NAME} -n {NAMESPACE}
    ```
  
5. Review the service instance or binding in SAP BTP to confirm it recognizes the re-established connection with the CR in your Kyma cluster.

If the connection is not re-established, ensure that your Kyma cluster's ID matches the cluster ID associated with the SAP BTP service instance or binding. View the cluster ID in the context details in the SAP BTP cockpit or by using the BTP CLI. If you discover mismatched IDs, reconfigure your Kyma cluster with the correct cluster ID.
