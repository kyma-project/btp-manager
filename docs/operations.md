| title                  |
|------------------------|
| BTP Manager Operations |

## Overview

BTP Manager performs the following operations:
- provisioning of SAP BTP Service Operator
- deprovisioning of SAP BTP Service Operator and its resources, Service Instances and Service Bindings

## Provisioning

### Prerequisites

The prerequisites for SAP BTP Service Operator provisioning are:
- Namespace `kyma-system`
- PriorityClass `kyma-system`
- Secret `sap-btp-manager` with data for SAP BTP Service Operator

Namespace and PriorityClass resources are created during Kyma installation. Secret is injected into the cluster by Kyma Environment Broker. If you want to provision SAP BTP Service Operator on a cluster without Kyma then you must create prerequisites yourself.

### Process

The provisioning process is a part of a module reconciliation and is carried out accordingly to the diagram presented below:

![Provisioning diagram](./assets/provisioning.svg)

Create a BtpOperator CR to trigger the reconciliation:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: BtpOperator
metadata:
  labels:
    app.kubernetes.io/part-of: btp-manager
  name: btpoperator
EOF
```

BtpOperator reconciler picks up the created CR and determines whether it should be responsible for representing the module status. BtpOperator CR reflects the status of operand, SAP BTP Service Operator, only when it is the oldest CR present in the cluster. In that case finalizer is added, CR is set to `Processing` state and the reconciliation proceeds. Otherwise, it is given an `Error` state with a status Condition containing details about CR responsible for reconciling the operand.

Next, the reconciler looks for a `sap-btp-manager` Secret in the `kyma-system` Namespace. This Secret contains Service Manager credentials for SAP BTP Service Operator and should be delivered to the cluster by KEB. If the Secret is missing, then an error is thrown, the reconciler sets `Error` state in the CR and stops the reconciliation until the Secret is created. When the Secret is present in the cluster then the reconciler verifies whether it contains required data. The Secret should contain the following keys: `clientid`, `clientsecret`, `sm_url`, `tokenurl`, `cluster_id`. None of the key values should be empty. If some required data is missing, the reconciler throws an error with the message about missing keys/values, sets the CR in `Error` state and stops the reconciliation until there is a change in the required Secret.

After checking the Secret, the reconciler prepares the module's chart for provisioning. It adds `app.kubernetes.io/managed-by: btp-manager` label to all chart resources, sets data from the required Secret as overrides and applies them among overrides from `values.yaml`. When the chart install info is correct, the reconciler starts the provisioning and waits specified time for all chart resources to be in `Ready` state. If timeout is reached then CR receives `Error` state and resources are checked again in the next reconciliation. The provisioning is successful when all chart resources are in `Ready` state and this is the condition which allows the reconciler to set CR in `Ready` state.

## Deprovisioning

To start the deprovisioning process, use the following command:

```
kubectl delete btpoperator your-btpoperator
```

The command triggers deletion of all service bindings, service instances and `your-btpoperator` on your cluster.

After the deletion of deployment and webhooks which `your-btpoperator` manages, the deprovisioning flow tries to perform deletion in hard delete mode. When it finds all service bindings and their secrets, and service instances across all namespaces, it tries to delete them.
The time limit for this operation is 20 minutes.
After this time, or in case of an error in hard deletion, the system goes into soft delete mode, which runs deletion of finalizers from service instances and bindings.
Regardless of mode, in the next step, all the SAP BTP service operator resources marked with the "managed-by:btp-operator" label are deleted.
If the process runs successfully, the finalizer on `your-btpoperator` itself is removed and the resource is deleted.
If an error occurs during deprovisioning, `your-btpoperator` is set to `Error`.

![Deprovisioning diagram](./assets/deprovisioning.svg)
