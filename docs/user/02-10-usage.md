---
title: Use BTP Manager to manage SAP BTP Service Operator 
---

You use BTP Manager to install or uninstall SAP BTP Service Operator.

You can install SAP BTP Service Operator: 
<details>
<summary>with a real BTP Manager Secret</summary>
<br>

To install SAP BTP Service Operator with a real BTP Manager Secret, follow these steps:
1. Create ServiceBinding to obtain the access credentials to the ServiceInstance as described in points 2b and 2c of the [Setup](https://github.com/SAP/sap-btp-service-operator#setup) section in the SAP BTP Service Operator documentation.
2. Copy the access credentials into the `hack/creds.json` file.
3. Call [`create-secret-file.sh`](../../hack/create-secret-file.sh). 
4. Apply the Secret in your cluster. 
 
   ```sh
   ./hack/create-secret-file.sh
   kubectl apply -f deployments/prerequisites.yaml
   kubectl apply -f hack/operator-secret.yaml
   kubectl apply -f examples/btp-operator.yaml
   ```
   </details>
  
<details>
<summary>with a dummy BTP Manager Secret</summary>
<br>

To install SAP BTP Service Operator with a dummy BTP Manager Secret, run the following commands:
```sh
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f examples/btp-manager-secret.yaml
kubectl apply -f examples/btp-operator.yaml
```
</details>
<br>

To check the `BtpOperator` CR status, run the following command:
```sh
kubectl get btpoperators btpoperator
```

The expected result is:
```
NAME                 STATE
btpoperator          Ready
```

### Uninstall SAP BTP Service Operator

To uninstall SAP BTP Service Operator, run the following commands:
```sh
kubectl delete -f examples/btp-operator.yaml
kubectl delete -f examples/btp-manager-secret.yaml
kubectl delete -f deployments/prerequisites.yaml
```
