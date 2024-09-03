# Create a Custom SAP BTP Service Operator Secret

1. Get the access credentials of the SAP Service Manager instance with the `service-operator-access` plan from its ServiceBinding. Copy them from the BTP cockpit as a JSON. 

2. Create the `creds.json` file in your working directory and save the credentials there.

3. In the same working directory, generate a Secret by calling the `create-secret-file.sh` script with the **operator** option as the first parameter and **your-secret-name**  as the second parameter.

    ```sh
    curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s operator 'my-secret'
    kubectl apply -f btp-access-credentials-secret.yaml
    ```
   <!-- this command also installs the Secret, right? - should it stay that way?-->