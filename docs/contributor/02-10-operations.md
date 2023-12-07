# BTP Manager Operations 

## Overview

BTP Manager performs the following operations:

- provisioning of the SAP BTP service operator
- update of the SAP BTP service operator
- deprovisioning of the SAP BTP service operator and its resources, ServiceInstances, and ServiceBindings

## Provisioning

### Prerequisites

The prerequisites for the SAP BTP service operator provisioning are:

- Namespace `kyma-system`
- PriorityClass `kyma-system`
- Secret `sap-btp-manager` with data for the SAP BTP service operator

The Namespace and PriorityClass resources are created during Kyma installation. The Secret is injected into the cluster
by Kyma Environment Broker (KEB). If you want to provision the SAP BTP service operator on a cluster without Kyma, you must create
the prerequisites yourself.

### Process

![Provisioning diagram](../assets/provisioning.svg)

The provisioning process is part of a module reconciliation. 
1. To trigger the reconciliation, create a [BtpOperator custom resource](../../api/v1alpha1/btpoperator_types.go) (CR):

   ```shell
   cat <<EOF | kubectl apply -f -
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: BtpOperator
   metadata:
     name: btpoperator
   EOF
   ```

2. The BtpOperator reconciler picks up the created CR and determines whether it should be responsible for representing the
module status. 
3. The BtpOperator CR reflects the status of the operand, that is, the SAP BTP service operator, only when it is
the oldest CR present in the cluster. Otherwise, it is given the `Error` state (3a) with the condition reason `OlderCRExists` and the message containing details about the CR responsible for reconciling the operand.
4. For the only or the oldest CR present in the cluster,  a finalizer is added, the CR is set to the `Processing` state, and the
reconciliation proceeds.
5. The reconciler looks for a `sap-btp-manager` Secret in the `kyma-system` Namespace with the label `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. This Secret contains the Service Manager credentials for the SAP BTP service operator and should be delivered to the cluster by KEB. If the Secret is missing, an error is thrown (5a), and the reconciler sets the `Warning` state (with the condition reason `MissingSecret`) in the CR and stops the reconciliation until the Secret
is created. 
6. When the Secret is present in the cluster, the reconciler verifies whether it contains the required data. The
Secret should contain the following keys: `clientid`, `clientsecret`, `sm_url`, `tokenurl`, `cluster_id`. None of the
key values should be empty. If some required data is missing, the reconciler throws an error (6a) with the message about
missing keys/values, sets the CR in the `Error` state (reason `InvalidSecret`), and stops the reconciliation until there is a change in the required
Secret.
7. After checking the Secret, the reconciler performs the apply and delete operations of the [module resources](../../module-resources).
One of GitHub Actions creates the `module-resources` directory, which contains manifests for applying and deleting operations. See [workflows](04-10-workflows.md#auto-update-chart-and-resources) for more details. First, the reconciler deletes outdated module resources stored as manifests in [to-delete.yml](../../module-resources/delete/to-delete.yml).
8. After all outdated resources are deleted successfully, the reconciler prepares current resources from manifests in the [apply](../../module-resources/apply) directory to be applied to the cluster.
The reconciler prepares certificates (regenerated if needed) and webhook configurations and adds these to the list of current resources. 
Then preparation of the current resources continues adding the `app.kubernetes.io/managed-by: btp-manager`, `chart-version: {CHART_VER}` labels to all module resources, 
setting `kyma-system` Namespace in all resources, setting module Secret and ConfigMap based on data read from the required Secret. 
9. After preparing the resources, the reconciler starts applying or updating them to the cluster. 
The non-existent resources are created using server-side apply to create the given resource, and the existent ones are updated.
10. The reconciler waits a specified time for all module resources existence in the cluster.
If the timeout is reached, the CR receives the `Error` state and the resources are rechecked in the next reconciliation. The reconciler has a fixed
set of [timeouts](../../controllers/btpoperator_controller.go) defined as `consts`, which limit the processing time
for performed operations. 
11. The provisioning is successful when all module resources exist in the cluster. This is the
condition that allows the reconciler to set the CR in the `Ready` state.

## Deprovisioning

![Deprovisioning diagram](../assets/deprovisioning.svg)

1. To start the deprovisioning process, use the following command:

   ```
   kubectl delete btpoperator {BTPOPERATOR_CR_NAME}
   ```

   The command triggers the deletion of the module resources in the cluster. By default, the existing ServiceInstances or ServiceBindings block the deletion. To unblock it, you must remove the existing ServiceInstances and ServiceBindings. Then, after the reconciliation, the SAP BTP Operator resource is gone.

   You can force the deletion by adding this label to the SAP BTP Operator resource:
   ```
   force-delete: "true"
   ```
   If you use the label, all the existing ServiceInstances and ServiceBindings are deleted automatically.

2. At first, the deprovisioning process tries to perform the deletion in a hard delete mode. It tries to delete all ServiceBindings and ServiceInstances across all Namespaces. The time limit for the hard delete is 20 minutes. 
3. Then it checks if there are any leftover ServiceBindings or ServiceInstances. 
4. The hard delete is unsuccessful if a timeout is reached, if some resources are still present, or in case of an error. Then the process goes into the soft delete mode.
5. The soft delete mode begins with deleting the SAP BTP service operator module deployment and webhooks.
6. The reconciler removes finalizers from ServiceBindings and deletes the related Secrets.
7. The reconciler checks if there are any ServiceBindings left.
8. Then it removes finalizers from ServiceInstances.
9. The last step in the soft delete mode is checking for any leftover ServiceInstances.
10. If any of steps 5-9 fail because of an error or unsuccessful resource deletion, the process throws a respective error, and the reconciliation starts again.
11. Regardless of the mode, all the SAP BTP service operator resources marked with the `app.kubernetes.io/managed-by:btp-manager` label are deleted. The deletion of module resources is based on resources GVKs (GroupVersionKinds) found in [manifests](../../module-resources). If the process succeeds, the finalizer on BtpOperator CR itself is removed, and the resource is deleted. If an error occurs during the deprovisioning (11a), the state of BtpOperator CR is set to `Error`.

## Conditions
The state of SAP BTP Operator CR is represented by [**Status**](https://github.com/kyma-project/module-manager/blob/main/pkg/declarative/v2/object.go#L23) that comprises State
and Conditions.
Only one Condition of type `Ready` is used.

[comment]: # (table_start)

| No.                  | CR state             | Condition type       | Condition status     | Condition reason                                | Remark                                                                                        |
| -------------------- | -------------------- | -------------------- | -------------------- | ----------------------------------------------- | --------------------------------------------------------------------------------------------- |
| 1                    | Ready                | Ready                | true                 | ReconcileSucceeded                              | Reconciled successfully                                                                       |
| 2                    | Ready                | Ready                | true                 | UpdateCheckSucceeded                            | Update not required                                                                           |
| 3                    | Ready                | Ready                | true                 | UpdateDone                                      | Update done                                                                                   |
| 4                    | Processing           | Ready                | false                | Initialized                                     | Initial processing or chart is inconsistent                                                   |
| 5                    | Processing           | Ready                | false                | Processing                                      | Final State after deprovisioning                                                              |
| 6                    | Processing           | Ready                | false                | UpdateCheck                                     | Checking for updates                                                                          |
| 7                    | Processing           | Ready                | false                | Updated                                         | Resource has been updated                                                                     |
| 8                    | Deleting             | Ready                | false                | HardDeleting                                    | Trying to hard delete                                                                         |
| 9                    | Deleting             | Ready                | false                | ServiceInstancesAndBindingsNotCleaned           | Deprovisioning blocked because of ServiceInstances and/or ServiceBindings existence           |
| 10                   | Deleting             | Ready                | false                | SoftDeleting                                    | Trying to soft delete after hard delete failed                                                |
| 11                   | Error                | Ready                | false                | ChartInstallFailed                              | Failure during chart installation                                                             |
| 12                   | Error                | Ready                | false                | ChartPathEmpty                                  | No chart path available for processing                                                        |
| 13                   | Error                | Ready                | false                | ConsistencyCheckFailed                          | Failure during consistency check                                                              |
| 14                   | Error                | Ready                | false                | DeletionOfOrphanedResourcesFailed               | Deletion of orphaned resources failed                                                         |
| 15                   | Error                | Ready                | false                | GettingConfigMapFailed                          | Getting Config Map failed                                                                     |
| 16                   | Error                | Ready                | false                | InconsistentChart                               | Chart is inconsistent. Reconciliation initialized                                             |
| 17                   | Error                | Ready                | false                | InvalidSecret                                   | sap-btp-manager secret does not contain required data - create proper secret                  |
| 18                   | Error                | Ready                | false                | OlderCRExists                                   | This CR is not the oldest one so does not represent the module State                          |
| 19                   | Error                | Ready                | false                | PreparingInstallInfoFailed                      | Error while preparing installation information                                                |
| 20                   | Error                | Ready                | false                | ProvisioningFailed                              | Provisioning failed                                                                           |
| 21                   | Error                | Ready                | false                | ReconcileFailed                                 | Reconciliation failed                                                                         |
| 22                   | Error                | Ready                | false                | ResourceRemovalFailed                           | Some resources can still be present due to errors while deprovisioning                        |
| 23                   | Error                | Ready                | false                | StoringChartDetailsFailed                       | Failure of storing chart details                                                              |
| 24                   | Warning              | Ready                | false                | MissingSecret                                   | sap-btp-manager secret was not found - create proper secret                                   |

[comment]: # (table_end)

## Updating

The update process is almost the same as the provisioning process. The only difference is BtpOperator CR existence in the cluster. 
For the update process, the CR should be present in the cluster with the `Ready` state.  
