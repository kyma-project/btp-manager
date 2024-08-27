# Working with Multiple Subaccounts

The SAP BTP Operator module supports multi-subaccount configurations in a single cluster.

To apply the multitenancy feature, choose the method that suits your needs and application architecture better: 
* Namespace-based mapping: Connect namespaces to separate subaccounts by configuring dedicated credentials for each namespace.
* Instance-level mapping: Define a specific subaccount for each service instance, regardless of the namespace context.

Both can be achieved through dedicated secrets managed in the `kyma-system` namespace.

### Namespace-Based Mapping

To connect a namespace to a specific subaccount, maintain access credentials to this subaccount in a Secret dedicated to the specific namespace. Define the `<namespace-name>-sap-btp-service-operator` Secret in the `kyma-system` namespace. 
See the examples of:
* Default access credentials:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: <namespace-name>-sap-btp-service-operator
    namespace: <centrally-managed-namespace>
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
1. Define a new Secret <!--? or Store access credentials?-->: Securely store access credentials for each subaccount in a separate Secret <!--or Secret resources?--> in the `kyma-system` namespace. <!--kyma-system??--> 
   See the examples of:
   * Default access credentials
      ```yaml
      apiVersion: v1
      kind: Secret
      metadata:
        name: <my-secret>
        namespace: <centrally managed namespace>
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
        namespace: <centrally managed namespace>
      type: Opaque
      stringData:
        clientid: <clientid>
        tls.crt: <certificate>
        tls.key: <key>
        sm_url: <sm_url>
        tokenurl: <auth_url>
        tokenurlsuffix: "/oauth/token"
      ```

2. Specify subaccount per service: Configure the Secret name in the ServiceInstance resource within the `btpAccessCredentialsSecret` property. The Secret containing the relevant subaccount's credentials explicitly tells SAP BTP Operator <!--??--> which subaccount to use to provision the service instance. The Secret must be located in the `kyma-system` namespace. 
    ```yaml
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceInstance
    metadata:
      name: sample-instance-1
    spec:
      serviceOfferingName: service-manager
      servicePlanName: subaccount-audit
      btpAccessCredentialsSecret: mybtpsecret
    ```

<!--or original: In the ServiceInstance resource, use the `btpAccessCredentialsSecret` property to reference the specific Secret containing the relevant subaccount's credentials. This explicitly tells SAP BTP Operator ?? which subaccount to use to provision the service instance.-->
SAP BTP Operator searches for the credentials in the following order:
1. Explicit Secret defined in a service instance
2. Default namespace Secret
3. Default cluster Secret
