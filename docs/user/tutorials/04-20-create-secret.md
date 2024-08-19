# Create and Install a Secret

<!--this content is for OS users only-->

To create a real BTP Manager Secret, follow these steps:
1. Create a ServiceBinding to obtain the access credentials to the ServiceInstance as described in the [Setup: Obtain the access credentials for the SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator#setup) section in the SAP BTP service operator documentation.
2. Copy and save the access credentials into your `creds.json` file in your working directory. 
3. In the same directory, run the following script to create the Secret:
   
   ```sh
   curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s
   ```

4. Apply the Secret in your cluster. 

   ```sh
   kubectl apply -f operator-secret.yaml
   ```

   > [!WARNING] 
   > The Secret already contains the required label: `app.kubernetes.io/managed-by: kcp-kyma-environment-broker`. Without this label, the Secret would not be visible to BTP Manager.

To check the `BtpOperator` custom resource (CR) status, run the following command:

```sh
kubectl get btpoperators btpoperator
```

The expected result is:

```
NAME                 STATE
btpoperator          Ready
```
