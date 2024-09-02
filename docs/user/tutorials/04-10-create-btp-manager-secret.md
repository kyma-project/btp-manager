# Create BTP Manager Secret

To create a real BTP Manager Secret, follow these steps:
1. Create a service binding to obtain the access credentials to the service instance as described in the [Setup: Obtain the access credentials for the SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator#setup) section in the SAP BTP service operator documentation.
2. Copy and save the access credentials into your `creds.json` file in your working directory. 
3. In the same directory, run the following script to create the Secret:
   
   ```sh
   curl https://raw.githubusercontent.com/kyma-project/btp-manager/main/hack/create-secret-file.sh | bash -s
   ```
   