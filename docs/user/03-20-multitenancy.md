# Working with Multiple Subaccounts

With the SAP BTP Operator module, you can create configurations for several subaccounts in a single Kyma cluster.

To apply the multitenancy feature, choose the method that suits your needs and application architecture: 
* Namespace-based mapping: Connect namespaces to separate subaccounts by configuring dedicated credentials for each namespace.
* Instance-level mapping: Define a specific subaccount for each service instance, regardless of the namespace context.

Regardless of the method, you must create Secrets managed in the `kyma-system` namespace.

### Namespace-Based Mapping

To connect a namespace to a specific subaccount, maintain access credentials to this subaccount in a Secret dedicated to the specific namespace. Define the `{NAMESPACE-NAME}-sap-btp-service-operator` Secret in the `kyma-system` namespace. 
See the following examples:
* Default access credentials:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {NAMESPACE_NAME}-sap-btp-service-operator
    namespace: kyma-system
  type: Opaque
  stringData:
    clientid: {CLIENT_ID}
    clientsecret: {CLIENT_SECRET}
    sm_url: {SM_URL}
    tokenurl: {AUTH_URL}
    tokenurlsuffix: "/oauth/token"
  ```

* mTLS access credentials:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {NAMESPACE_NAME}-sap-btp-service-operator
    namespace: kyma-system
  type: Opaque
  stringData:
    clientid: {CLIENT_ID}
    tls.crt: {TLS_CERTIFICATE}
    tls.key: {TLS_KEY}
    sm_url: {SM_URL}
    tokenurl: {AUTH_URL}
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
        name: {SECRET_NAME}
        namespace: kyma-system
      type: Opaque
      stringData:
        clientid: {CLIENT_ID}
        clientsecret: {CLIENT_SECRET}
        sm_url: {SM_URL}
        tokenurl: {AUTH_URL}
        tokenurlsuffix: "/oauth/token"
      ```
    * mTLS access credentials
      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        name: {SECRET_NAME}
        namespace: kyma-system
      type: Opaque
      stringData:
        clientid: {CLIENT_ID}
        tls.crt: {TLS_CERTIFICATE}
        tls.key: {TLS_KEY}
        sm_url: {SM_URL}
        tokenurl: {AUTH_URL}
        tokenurlsuffix: "/oauth/token"
      ```

2. Specify a subaccount per service: Configure the Secret name in the ServiceInstance resource within the **btpAccessCredentialsSecret** property. The Secret containing the relevant subaccount's credentials tells SAP BTP Operator explicitly which subaccount to use to provision the service instance. The Secret must be located in the `kyma-system` namespace.
    ```yaml
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceInstance
    metadata:
      name: {SERVICE_INSTANCE_NAME}
    spec:
      serviceOfferingName: service-manager
      servicePlanName: subaccount-audit
      btpAccessCredentialsSecret: {SECRET_NAME}
    ```
### Secrets Precedence

SAP BTP Operator searches for the credentials in the following order:
1. Explicit Secret defined in a service instance
2. Default namespace Secret
3. Default cluster Secret
