# Create a Custom `sap-btp-service-operator` Secret

1. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its service binding. Copy them from the BTP cockpit as a JSON.

2. Create the `creds.json` file in your working directory and save the credentials there.

3. In the same working directory, generate the Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **your-secret-name**  as the second parameter.

    ```sh
    curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s operator 'my-secret'
    ```

The expected result is the file `btp-access-credentials-secret.yaml` created in your working directory:

```yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: {YOUR_SECRET_NAME}
  namespace: kyma-system
data:
  clientid: {CLIENT_ID}
  clientsecret: {CLIENT_SECRET}
  sm_url: {SM_URL}
  tokenurl: {AUTH_URL}
  tokenurlsuffix: "/oauth/token"
```

## Next Steps

To apply the Secret in your cluster, run:

```sh
kubectl apply -f btp-access-credentials-secret.yaml
```

> [!TIP]
> When you do not intend to use your custom Secret any more, delete it with this command:
> ```bash
> kubectl delete secret {YOUR_SECRET_NAME} -n kyma-system
>  ```
