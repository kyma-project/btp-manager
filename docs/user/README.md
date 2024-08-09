# SAP BTP Operator Module

Learn more about the SAP BTP Operator module. Use it to enable Service Management and use SAP BTP services from your Kyma cluster.

## What is SAP BTP Operator?

The SAP BTP Operator module provides Service Management, which allows you to consume [SAP BTP services](https://discovery-center.cloud.sap/protected/index.html#/viewServices) from your Kyma cluster using Kubernetes-native tools.
Within the SAP BTP Operator module, [BTP Manager](https://github.com/kyma-project/btp-manager) installs the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator/blob/main/README.md).
The SAP BTP service operator enables provisioning and managing Service Instances and Service Bindings of SAP BTP services so that your Kubernetes-native applications can access and use the services from your cluster.

## Features

The SAP BTP Operator module provides the following features:
* Preconfigured and managed Secret for your cluster: Your Secret is readily available on Kyma instance creation.
* Managing access: <!--add what you can do-->
* Multitenancy: You can configure multiple subaccounts in a single cluster.
* Lifecycle management of Service Instances and Service Bindings: You can create and delete Service Instances and Service Bindings. 
* Service Binding rotation: You can enhance security by automatically rotating the credentials associated with your Service Bindings.
* Formatting Service Binding Secret:
* Subaccount for a ServiceInstance resource: You can deploy Service Instances belonging to different subaccounts within the same namespace.

## Scope  

By default, the scope of the SAP BTP Operator module is your Kyma runtime, which consumes SAP BTP services from the subaccount in which you created it. You can extend the scope of the module by adding more subaccounts and consuming services from them in one cluster. Depending on the number of configured Secrets, the scope can range from one to multiple subaccounts.

## Architecture

The SAP BTP Operator module allows for the initial provisioning and retrieving credentials necessary for the application to use a SAP BTP service.

![SAP BTP Operator architecture](../assets/BtpOperator_architecture.drawio.svg) <!-- the arrow to the module or to the service operator?-->

### SAP BTP, Kyma Runtime

SAP BTP, Kyma runtime is <!--runs on(?)--> a Kubernetes cluster that wraps the SAP BTP Operator module, API server and one or more applications. The application or multiple applications are the actual consumers of SAP BTP services.

### BTP Manager

BTP Manager is an operator based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator CustomResourceDefinition (CRD)](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml). 
BTP Manager performs the following operations:
* manages the lifecycle of the SAP BTP service operator, including provisioning the latest version, updating, and deprovisioning
* configures the SAP BTP service operator

### SAP BTP Service Operator

The SAP BTP service operator allows you to connect SAP BTP services to your cluster and then manage them using Kubernetes-native tools. It is responsible for communicating with SAP Service Manager. The operator's resources, Service Instances and Service Bindings, come from the `services.cloud.sap.com` API group.

### SAP Service Manager

[SAP Service Manager](https://help.sap.com/docs/service-manager/sap-service-manager/sap-service-manager?locale=en-US) is the central registry for service brokers and platforms in SAP BTP, which allows you to:
* consume platform services in any connected runtime environment
* track the creation and management of Service Instances
* share services and Service Instances between different runtimes

SAP Service Manager uses [Open Service Broker API](https://www.openservicebrokerapi.org/) (OSB API) to communicate with service brokers.

### Service Brokers

Service Brokers manage the lifecycle of services. SAP Service Manager interacts with service brokers using OSB API to provision and manage Service Instances and Service Bindings.

## API / Custom Resource Definitions

The `btpoperators.operator.kyma-project.io` Custom Resource Definition (CRD) describes the kind and the format of data that SAP BTP Operator <!--or is it BTP Manager?--> uses to configure resources.

See the documentation related to the BtpOperator custom resource (CR):
* SAP BTP Operator
* Sercice Instance
* Service Binding
<!--add links to the new doc(s)-->

## Resource Consumption

To learn more about the resources used by the SAP BTP Operator module, see [Kyma Modules' Sizing](https://help.sap.com/docs/btp/sap-business-technology-platform-internal/kyma-modules-sizing?locale=en-US&state=DRAFT&version=Internal#sap-btp-operator).
