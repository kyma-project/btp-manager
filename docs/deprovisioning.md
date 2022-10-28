## Deprovisioning

To start deprovisioning proccess use:

```
kubectl delete btpoperator yourbtpoperator
```

This command will trigger deletion of all service bindings, service instances and btp operator resources on your cluster.

At first, the deletion of deployment and webhooks which yourbtpoperator manages will happen, then deprovisioning flow will try to perform deletion in hard delete mode, which mean it will find all service bindings(and their secrets) and service instances across all namespaces and will try to make kubernetes delete on it.
The time limit for this operation is 20 minutes.
After this time, or in case of error in hard deletion, the system will go into soft delete mode which mean deletion of finalizers from service instances and bindings.
Regardless of mode, at later step, the all btp operator resources marked with label "managed-by:btp-operator" are deleted.
If proccess pass with success then finally finalizer on yourbtpoperator itself is removed and the resource is deleted.
If any error will happen during deprovisioning, then yourbtpoperator will be set to Error state.

![KEB diagram](./assets/keb-architecture.svg)
