# SAP BTP Operator Custom Resource

The `btpoperators.operator.kyma-project.io` Custom Resource Definition (CRD) is a comprehensive specification that defines the structure and format used to manage the configuration and status of the SAP BTP Operator module within your Kyma environment.

To get the latest CRD in the YAML format, run the following command:

```shell
kubectl get crd btpoperators.operator.kyma-project.io -o yaml
```
You can only have one SAP BTP Operator (BtpOperator) CR. If multiple BtpOperator CRs exist in the cluster, the oldest one reconciles the module. An additional BtpOperator CR has the `Warning` state.

## Sample Custom Resource

The following BtpOperator object defines a module:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: BtpOperator
metadata:
  finalizers:
    - operator.kyma-project.io/btp-manager
  labels:
    app.kubernetes.io/created-by: btp-manager
    app.kubernetes.io/instance: btpoperator
    app.kubernetes.io/managed-by: btp-manager
    app.kubernetes.io/name: btpoperator
    app.kubernetes.io/part-of: btp-manager
  name: btpoperator
  namespace: kyma-system
spec: {}
status:
  conditions:
    - lastTransitionTime: '2024-08-08T14:39:01Z'
      message: Module provisioning succeeded
      reason: ReconcileSucceeded
      status: 'True'
      type: Ready
  state: Ready
```

## Custom Resource Parameters

**Spec:** 
<!-- is this section corrct?-->
Currently, no entry parameters are available for configuration in the BtpOperator CR.

**Status:**

| No.        | CR state             | Condition type       | Condition status     | Condition reason                                | Description                                                                                |
| ---------- | -------------------- | -------------------- | -------------------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------ |
| 1          | Ready                | Ready                | true                 | ReconcileSucceeded                              | Reconciled successfully                                                                    |
| 2          | Ready                | Ready                | true                 | UpdateCheckSucceeded                            | Update not required                                                                        |
| 3          | Ready                | Ready                | true                 | UpdateDone                                      | Update done                                                                                |
| 4          | Processing           | Ready                | false                | Initialized                                     | Initial processing or chart is inconsistent                                                |
| 5          | Processing           | Ready                | false                | Processing                                      | Final State after deprovisioning                                                           |
| 6          | Processing           | Ready                | false                | UpdateCheck                                     | Checking for updates                                                                       |
| 7          | Processing           | Ready                | false                | Updated                                         | Resource has been updated                                                                  |
| 8          | Deleting             | Ready                | false                | HardDeleting                                    | Trying to hard delete                                                                      |
| 9          | Deleting             | Ready                | false                | SoftDeleting                                    | Trying to soft delete after hard delete failed                                             |
| 10         | Warning              | Ready                | false                | ServiceInstancesAndBindingsNotCleaned           | Deprovisioning blocked because of service instances and/or service bindings existence      |
| 11         | Warning              | Ready                | false                | OlderCRExists                                   | This CR is not the oldest one, so does not represent the module State                       |
| 12         | Warning              | Ready                | false                | MissingSecret                                   | `sap-btp-manager` Secret was not found - create proper Secret                              |
| 13         | Error                | Ready                | false                | ChartInstallFailed                              | Failure during chart installation                                                          |
| 14         | Error                | Ready                | false                | ChartPathEmpty                                  | No chart path available for processing                                                     |
| 15         | Error                | Ready                | false                | ConsistencyCheckFailed                          | Failure during consistency check                                                           |
| 16         | Error                | Ready                | false                | DeletionOfOrphanedResourcesFailed               | Deletion of orphaned resources failed                                                      |
| 17         | Error                | Ready                | false                | GettingConfigMapFailed                          | Getting Config Map failed                                                                  |
| 18         | Error                | Ready                | false                | InconsistentChart                               | Chart is inconsistent. Reconciliation initialized                                          |
| 19         | Error                | Ready                | false                | InvalidSecret                                   | `sap-btp-manager` Secret does not contain required data - create proper Secret             |
| 20         | Error                | Ready                | false                | PreparingInstallInfoFailed                      | Error while preparing installation information                                             |
| 21         | Error                | Ready                | false                | ProvisioningFailed                              | Provisioning failed                                                                        |
| 22         | Error                | Ready                | false                | ReconcileFailed                                 | Reconciliation failed                                                                      |
| 23         | Error                | Ready                | false                | ResourceRemovalFailed                           | Some resources can still be present due to errors while deprovisioning                     |
| 24         | Error                | Ready                | false                | StoringChartDetailsFailed                       | Failure of storing chart details                                                           |

> [!NOTE]
> If an operation returns the `Warning` CR state, it has encountered a problem. Read the relevant description in the table and take action to solve the problem.