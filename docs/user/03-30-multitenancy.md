# Working with Multiple Subaccounts

With the SAP BTP Operator module, you can create configurations for several subaccounts in a single Kyma cluster.

To apply the multitenancy feature, choose the method that suits your needs and application architecture: 
* Namespace-based mapping: Connect namespaces to separate subaccounts by configuring dedicated credentials for each namespace.
* Instance-level mapping: Define a specific subaccount for each service instance, regardless of the namespace context.

Regardless of the method, you must create Secrets managed in the `kyma-system` namespace.

### Namespace-Based Mapping

To connect a namespace to a specific subaccount, maintain access credentials to this subaccount in a Secret dedicated to the specific namespace. Define the `<namespace-name>-sap-btp-service-operator` Secret in the `kyma-system` namespace. 
See the following examples:
* Default access credentials:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: <namespace-name>-sap-btp-service-operator
    namespace: kyma-system
  type: Opaque
  stringData:
    clientid: "<clientid>"
    clientsecret: "<clientsecret>"
    sm_url: "<sm_url>"
    tokenurl: "<auth_url>"
    tokenurlsuffix: "/oauth/token"
  ```

* mTLS access credentials:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: <namespace-name>-sap-btp-service-operator
    namespace: kyma-system
  type: Opaque
  stringData:
    clientid: <clientid>
    tls.crt: <certificate>
    tls.key: <key>
    sm_url: <sm_url>
    tokenurl: <auth_url>
    tokenurlsuffix: "/oauth/token"
  ```

### Instance-Level Mapping

To deploy service instances belonging to different subaccounts within the same namespace, follow these steps:
1. Define a new Secret: Securely store access credentials for each subaccount in a separate Secret in the `kyma-system` namespace. 
   See the following examples:
   * Default access credentials
      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        name: <my-secret>
        namespace: kyma-system
      type: Opaque
      stringData:
        clientid: "<clientid>"
        clientsecret: "<clientsecret>"
        sm_url: "<sm_url>"
        tokenurl: "<auth_url>"
        tokenurlsuffix: "/oauth/token"
      ```
    * mTLS access credentials
      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        name: <my-secret>
        namespace: kyma-system
      type: Opaque
      stringData:
        clientid: <clientid>
        tls.crt: <certificate>
        tls.key: <key>
        sm_url: <sm_url>
        tokenurl: <auth_url>
        tokenurlsuffix: "/oauth/token"
      ```

2. Specify a subaccount per service: Configure the Secret name in the ServiceInstance resource within the `btpAccessCredentialsSecret` property. The Secret containing the relevant subaccount's credentials tells SAP BTP Operator explicitly which subaccount to use to provision the service instance. The Secret must be located in the `kyma-system` namespace. 
    ```yaml
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceInstance
    metadata:
      name: my-instance-1
    spec:
      serviceOfferingName: service-manager
      servicePlanName: subaccount-audit
      btpAccessCredentialsSecret: mybtpsecret
    ```
### Secrets Precedence

SAP BTP Operator searches for the credentials in the following order:
1. Explicit Secret defined in a service instance
2. Default namespace Secret
3. Default cluster Secret
