# BTP Manager operations 

## Overview

BTP Manager performs the following operations:

- provisioning of SAP BTP Service Operator
- update of SAP BTP Service Operator
- deprovisioning of SAP BTP Service Operator and its resources, Service Instances, and Service Bindings

## Provisioning

### Prerequisites

The prerequisites for SAP BTP Service Operator provisioning are:

- Namespace `kyma-system`
- PriorityClass `kyma-system`
- Secret `sap-btp-manager` with data for SAP BTP Service Operator

The Namespace and PriorityClass resources are created during Kyma installation. The Secret is injected into the cluster
by Kyma Environment Broker (KEB). If you want to provision SAP BTP Service Operator on a cluster without Kyma, you must create
the prerequisites yourself.

### Process

![Provisioning diagram](/docs/assets/deprovisioning.svg)

The provisioning process is part of a module reconciliation. To trigger the reconciliation, create a [BtpOperator CR](/api/v1alpha1/btpoperator_types.go):

```shell
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: BtpOperator
metadata:
  name: btpoperator
  labels:
    app.kubernetes.io/managed-by: btp-manager
EOF
```

The BtpOperator reconciler picks up the created CR and determines whether it should be responsible for representing the
module status. The BtpOperator CR reflects the status of the operand, that is, SAP BTP Service Operator, only when it is
the oldest CR present in the cluster. In that case a finalizer is added, the CR is set to `Processing` state and the
reconciliation proceeds.
Otherwise, it is given an `Error` state with the condition reason `OlderCRExists` and message containing details
about the CR responsible for reconciling the operand.

Next, the reconciler looks for a `sap-btp-manager` Secret in the `kyma-system` Namespace with the label `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. This Secret contains the Service
Manager credentials for SAP BTP Service Operator and should be delivered to the cluster by KEB. If the Secret is
missing, an error is thrown, the reconciler sets `Warning` state (with the condition reason `MissingSecret`) in the CR and stops the reconciliation until the Secret
is created. When the Secret is present in the cluster, the reconciler verifies whether it contains required data. The
Secret should contain the following keys: `clientid`, `clientsecret`, `sm_url`, `tokenurl`, `cluster_id`. None of the
key values should be empty. If some required data is missing, the reconciler throws an error with the message about
missing keys/values, sets the CR in `Error` state (reason `InvalidSecret`), and stops the reconciliation until there is a change in the required
Secret.

After checking the Secret, the reconciler proceeds to apply and delete operations of the [module resources](/module-resources).
The `module-resources` directory is created by one of GitHub Actions and contains manifests for applying and deleting operations. See [workflows](04-10-workflows.md#auto-update-chart-and-resources) for more details.
First, the reconciler deletes outdated module resources stored as manifests in [to-delete.yml](/module-resources/delete/to-delete.yml).
When all outdated resources are deleted successfully, the reconciler prepares current resources from manifests in the [apply](/module-resources/apply) directory to be applied to the cluster.
The reconciler prepares certificates (regenerated if needed) and webhook configurations and adds these to the list of current resources. 
Then preparation of the current resources continues adding the `app.kubernetes.io/managed-by: btp-manager`, `chart-version: {CHART_VER}` labels to all module resources, 
setting `kyma-system` Namespace in all resources, setting module Secret and ConfigMap based on data read from the required Secret. 
After preparing the resources, the reconciler starts applying or updating them to the cluster. 
The non-existent resources are created using server-side apply to create the given resource, the existent ones are updated.
The reconciler waits a specified time for all module resources existence in the cluster.
If the timeout is reached, the CR receives the `Error` state and the resources are checked again in the next reconciliation. The reconciler has a fixed
set of [timeouts](/controllers/btpoperator_controller.go) defined as `consts` which limit the processing time
for performed operations. The provisioning is successful when all module resources exist in the cluster. This is the
condition which allows the reconciler to set the CR in `Ready` state.

## Deprovisioning

![Deprovisioning diagram](/docs/assets/deprovisioning.svg)

To start the deprovisioning process, use the following command:

```
kubectl delete btpoperator {BTPOPERATOR_CR_NAME}
```

The command triggers the deletion of the module resources in the cluster. By default, the existing Service Instances or Service Bindings block the deletion. To unblock it, you must remove the existing Service Instances and Service Bindings. Then, after a maximum of 15-minute reconciliation time, the BTP Operator resource is gone.

You can force the deletion by adding this label to the BTP Operator resource:
```
force-delete: "true"
```
If you use the label, all the existing Service Instances and Service Bindings are deleted automatically.

At first, the deprovisioning process tries to perform the deletion in a hard delete mode. It tries to delete all 
Service Bindings and Service Instances across all Namespaces. The time limit for the hard delete is 20 minutes. 
After this time, or in case of an error, the process goes into soft delete mode, which runs deletion of finalizers from Service Instances and Service Bindings.

In order to delete finalizers the reconciler deletes module deployment and webhooks.
Regardless of mode, in the next step, all SAP BTP Service Operator resources marked with the `app.kubernetes.io/managed-by:btp-manager`
label are deleted. The deletion process of module resources is based on resources GVKs (GroupVersionKinds) found in [manifests](/module-resources).
If the process succeeds, the finalizer on BtpOperator CR itself is removed and the resource is deleted.
If an error occurs during the deprovisioning, state of BtpOperator CR is set to `Error`. 

## Conditions
The state of BTP Operator CR is represented by [**Status**](https://github.com/kyma-project/module-manager/blob/main/pkg/declarative/v2/object.go#L23) that comprises State
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

The update process is almost the same as the provisioning process. The only difference is BtpOperator CR existence in the cluster, 
for the update process the custom resource should be present in the cluster with `Ready` state.  
