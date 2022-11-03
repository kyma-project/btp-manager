## Deprovisioning

To start the deprovisioning process, use the following command:

```
kubectl delete btpoperator your-btpoperator
```

The command triggers deletion of all service bindings, service instances and BtpOperator resources on your cluster.

After the deletion of deployment and webhooks which `your-btpoperator` manages, the deprovisioning flow tries to perform deletion in hard delete mode. When it finds all service bindings and their secrets, and service instances across all namespaces, it tries to delete Kubernetes.
The time limit for this operation is 20 minutes.
After this time, or in case of an error in hard deletion, the system goes into soft delete mode, which runs deletion of finalizers from service instances and bindings.
Regardless of mode, in the next step, the all BtpOperator resources marked with the "managed-by:btp-operator" label are deleted.
If the process runs successfully, the finalizer on `your-btpoperator` itself is removed and the resource is deleted.
If an error occurs during deprovisioning, `your-btpoperator` is set to `Error`.

![Deprovisioning diagram](./assets/deprovisioning.svg)
