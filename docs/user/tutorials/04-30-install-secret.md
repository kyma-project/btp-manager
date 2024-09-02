# Install a BTP Manager Secret

<!--this content is for OS users only-->
After successfully creating your Secret, install it in your cluster.

> [!TIP]
> For instructions on creating a BTP Manager Secret, see the [dedicated tutorial](04-10-create-btp-manager-secret.md).

1. To apply the Secret in your cluster, run the following command: 

   ```sh
   kubectl apply -f operator-secret.yaml
   ```

   > [!WARNING] 
   > The Secret already contains the required label: `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. Without this label, the Secret is not visible to BTP Manager.

2. To check the `BtpOperator` custom resource (CR) status, run the following command:

   ```sh
   kubectl get btpoperators btpoperator
   ```

   The expected result is:

   ```
   NAME                 STATE
   btpoperator          Ready
   ```
