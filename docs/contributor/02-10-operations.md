# BTP Manager Operations

## Overview

BTP Manager performs the following operations:

* Provisions the SAP BTP service operator
* Updates the SAP BTP service operator
* Deprovisions the SAP BTP service operator, its ServiceInstance resources, and ServiceBinding resources

## Provisioning

### Prerequisites

* The `kyma-system` namespace
* The `sap-btp-manager` Secret with data for the SAP BTP service operator

The `kyma-system` namespace is created during SAP BTP, Kyma runtime installation. The Secret is injected into the cluster by Kyma Environment Broker (KEB).
If you want to provision the SAP BTP service operator in a cluster without Kyma runtime, you must create the prerequisites yourself.

### Process

![Provisioning diagram](../assets/provisioning.drawio.svg)

The provisioning process is part of a module reconciliation.

1. To trigger reconciliation, create a [BtpOperator custom resource (CR)](https://github.com/kyma-project/btp-manager/blob/main/api/v1alpha1/btpoperator_types.go).

   ```shell
   cat <<EOF | kubectl apply -f -
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: BtpOperator
   metadata:
     name: btpoperator
     namespace: kyma-system
   EOF
   ```

2. The BtpOperator reconciler picks up the created CR and determines whether the CR should be responsible for representing the module status.
3. The BtpOperator CR reflects the status of the SAP BTP service operator only when it is in the `kyma-system` namespace and has the required name. Otherwise, it is set to the `Warning` state with the condition reason `WrongNamespaceOrName` (3a).
4. For the only valid CR present in the cluster, a finalizer is added, the CR is set to the `Processing` state, and reconciliation continues.
5. In the `kyma-system` namespace, the reconciler looks for the `sap-btp-manager` Secret with the label `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. This Secret contains the SAP Service Manager credentials for the SAP BTP service operator and is delivered to the cluster by KEB. If the Secret is missing, an error is thrown (5a). The reconciler sets the `Warning` state (reason `MissingSecret`) in the CR, and stops reconciliation. New reconciliation is queued and processed after some time, or is triggered by changing the Secret.
6. If the Secret exists in the cluster, the reconciler checks for the following required data: **clientid**, **clientsecret**, **sm_url**, **tokenurl**, **cluster_id**. All the keys must have values.
   If any required data is missing, the reconciler throws an error (6a) and sets the CR to `Error` (reason `InvalidSecret`) until the required Secret is updated.
7. The reconciler performs the apply and delete operations of the [module resources](https://github.com/kyma-project/btp-manager/tree/main/module-resources).
   One of GitHub Actions creates the `module-resources` directory, which contains manifests for applying and deleting operations. For more details, see the [Auto Update Chart and Resources](https://github.com/kyma-project/btp-manager/blob/main/docs/contributor/04-10-workflows.md#auto-update-chart-and-resources) workflow. The reconciler deletes outdated module resources stored as manifests in [to-delete.yml](https://github.com/kyma-project/btp-manager/blob/main/module-resources/delete/to-delete.yml).
8. After outdated resources are deleted, the reconciler prepares current resources from manifests in the [apply](https://github.com/kyma-project/btp-manager/tree/main/module-resources/apply) directory.
   The reconciler prepares certificates (regenerated if needed) and webhook configurations, and adds them to the list of current resources.
   Preparation of the current resources continues by adding the `app.kubernetes.io/managed-by: btp-manager` and `chart-version: {CHART_VER}` labels to all module resources, setting the `kyma-system` namespace in all resources, setting the module Secret and ConfigMap based on the data read from the required Secret. The reconciler also sets the SAP BTP service operator's Deployment images by reading the images from the **SAP_BTP_SERVICE_OPERATOR** and **KUBE_RBAC_PROXY** environment variables, and setting appropriate **image** fields in the Deployment's `spec`.
9. When the resources are prepared, the reconciler starts applying or updating them to the cluster.
   The missing resources are created using server-side apply to create a given resource and the existing ones are updated.
10. The reconciler waits a specified period for all module resources to exist in the cluster.
   If the timeout is reached, the CR is set to `Error`, and resources are rechecked in the next reconciliation.
   The reconciler has a fixed set of [timeouts](https://github.com/kyma-project/btp-manager/blob/main/controllers/btpoperator_controller.go) defined as `consts`, which limit the processing time for performed operations.
11. When all module resources exist in the cluster, provisioning is successful, and the reconciler can set the CR in the `Ready` state.

## Deprovisioning

![Deprovisioning diagram](../assets/deprovisioning.drawio.svg)

1. To start the deprovisioning process, run the following command:

   ```
   kubectl delete btpoperator {BTPOPERATOR_CR_NAME}
   ```

   The command triggers deletion of module resources in the cluster. By default, existing service instances or service bindings block the deletion. To unblock it, you must remove these resources. Then, after the reconciliation, the SAP BTP Operator resource is gone.

   To force deletion, add this label to the SAP BTP Operator resource:

   ```
   force-delete: "true"
   ```

   With this label, all the existing service instances and service bindings are deleted automatically.

2. The deprovisioning process tries to perform the deletion in hard-delete mode. It tries to delete all service bindings and service instances across all namespaces. The time limit for the hard delete is 20 minutes. 
3. Then, it checks if there are any leftover service bindings or service instances.
4. If a timeout is reached, if some resources are still present, or in case of an error, the hard delete is unsuccessful. The process goes into soft-delete mode.
5. Soft-delete mode begins with deleting the SAP BTP service operator Deployment and webhooks.
6. The reconciler removes finalizers from service bindings and deletes the related Secrets.
7. The reconciler checks if there are any service bindings left.
8. The reconciler removes finalizers from service instances.
9. The last step in soft-delete mode is checking for any leftover service instances.
10. If any soft-delete step fails because of an error or unsuccessful resource deletion, the process throws a respective error, and the reconciliation starts again.
11. Regardless of the mode, all SAP BTP service operator resources marked with the `app.kubernetes.io/managed-by:btp-manager` label are deleted. Deletion of module resources is based on resources GVKs (GroupVersionKinds) in [manifests](../../module-resources). If the process succeeds, the finalizer on the BtpOperator CR is removed, and the resource is deleted. If an error occurs during deprovisioning (11a), the BtpOperator CR is set to `Error`.

## Conditions
The state of the SAP BTP Operator CR is represented by [**Status**](https://github.com/kyma-project/module-manager/blob/main/pkg/declarative/v2/object.go#L23),
which comprises state and condition.
The only condition used is of type `Ready`.

[comment]: # (table_start)

| No. | CR state             | Condition type       | Condition status     | Condition reason                                            | Remark                                                                                        |
|-----| -------------------- | -------------------- | -------------------- | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| 1   | Ready                | Ready                | true                 | ReconcileSucceeded                                          | Reconciled successfully                                                                       |
| 2   | Ready                | Ready                | true                 | UpdateCheckSucceeded                                        | Update not required                                                                           |
| 3   | Ready                | Ready                | true                 | UpdateDone                                                  | Update done                                                                                   |
| 4   | Processing           | Ready                | false                | ClusterIdChanged                                            | Cluster ID changed                                                                            |
| 5   | Processing           | Ready                | false                | CredentialsNamespaceChanged                                 | Credentials namespace changed                                                                 |
| 6   | Processing           | Ready                | false                | Initialized                                                 | Initial processing or chart is inconsistent                                                   |
| 7   | Processing           | Ready                | false                | Processing                                                  | Final State after deprovisioning                                                              |
| 8   | Processing           | Ready                | false                | UpdateCheck                                                 | Checking for updates                                                                          |
| 9   | Processing           | Ready                | false                | Updated                                                     | Resource has been updated                                                                     |
| 10  | Deleting             | Ready                | false                | HardDeleting                                                | Trying to hard delete                                                                         |
| 11  | Deleting             | Ready                | false                | SoftDeleting                                                | Trying to soft-delete after hard-delete failed                                                |
| 12  | Error                | Ready                | false                | AnnotatingSecretFailed                                      | Annotating the required Secret failed                                                         |
| 13  | Error                | Ready                | false                | ChartInstallFailed                                          | Failure during chart installation                                                             |
| 14  | Error                | Ready                | false                | ChartPathEmpty                                              | No chart path available for processing                                                        |
| 15  | Error                | Ready                | false                | ConsistencyCheckFailed                                      | Failure during consistency check                                                              |
| 16  | Error                | Ready                | false                | DeletionOfOrphanedResourcesFailed                           | Deletion of orphaned resources failed                                                         |
| 17  | Error                | Ready                | false                | GettingConfigMapFailed                                      | Getting ConfigMap failed                                                                      |
| 18  | Error                | Ready                | false                | GettingDefaultCredentialsSecretFailed                       | Getting default credentials Secret failed                                                     |
| 19  | Error                | Ready                | false                | GettingSapBtpServiceOperatorClusterIdSecretFailed           | Getting SAP BTP service operator Cluster ID Secret failed                                     |
| 20  | Error                | Ready                | false                | GettingSapBtpServiceOperatorConfigMapFailed                 | Getting SAP BTP service operator ConfigMap failed                                             |
| 21  | Error                | Ready                | false                | InconsistentChart                                           | Chart is inconsistent, reconciliation initialized                                             |
| 22  | Error                | Ready                | false                | InvalidSecret                                               | `sap-btp-manager` Secret does not contain required data - create proper Secret                |
| 23  | Error                | Ready                | false                | PreparingInstallInfoFailed                                  | Error while preparing installation information                                                |
| 24  | Error                | Ready                | false                | ProvisioningFailed                                          | Provisioning failed                                                                           |
| 25  | Error                | Ready                | false                | ReconcileFailed                                             | Reconciliation failed                                                                         |
| 26  | Error                | Ready                | false                | ResourceRemovalFailed                                       | Some resources can still be present due to errors while deprovisioning                        |
| 27  | Error                | Ready                | false                | StoringChartDetailsFailed                                   | Failure of storing chart details                                                              |
| 28  | Warning              | Ready                | false                | MissingSecret                                               | `sap-btp-manager` Secret was not found - create proper Secret                                 |
| 29  | Warning              | Ready                | false                | ServiceInstancesAndBindingsNotCleaned                       | Deprovisioning blocked because of ServiceInstances and/or ServiceBindings existence           |
| 30  | Warning              | Ready                | false                | WrongNamespaceOrName                                        | Wrong namespace or name                                                                       |

[comment]: # (table_end)

## Updating

The update process is almost the same as the provisioning process. The only difference is the BtpOperator CR's existence in the cluster. 
For the update process, the CR should be present in the cluster with a `Ready` state.
