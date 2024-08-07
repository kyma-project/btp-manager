# SAP BTP Operator Module

Learn more about the SAP BTP Operator module. Use it to enable Service Management and use SAP BTP services from your Kyma cluster.

## What is SAP BTP Operator?

 The SAP BTP Operator module provides Service Management, which allows you to consume [SAP BTP services](https://platformx-d8bd51250.dispatcher.us2.hana.ondemand.com/protected/index.html#/viewServices) <!-- this is a link to Demo Environment; use the one to the prod version in HP; make sure that's ok --> from your Kyma cluster using Kubernetes-native tools.
Within the SAP BTP Operator module, [BTP Manager](https://github.com/kyma-project/btp-manager) installs the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator/blob/main/README.md).
The SAP BTP service operator enables provisioning and managing Service Instances and Service Bindings of SAP BTP services so that your Kubernetes-native applications can access and use the services from your cluster.

## Features

The SAP BTP Operator module provides the following features:
* Preconfigured Secret for your cluster: Your Secret is readily available on Kyma instance creation.
* Recreation of the Secret: Your Secret is automatically recreated in case of unintentional <!--acccidental?--> deletion.
* Access to the Secret in the SAP BTP cockpit
* Multitenancy: You can configure multiple subaccounts in a single cluster.
* Service Binding rotation: You can enhance security by automatically rotating the credentials associated with your Service Bindings.
* Subaccount for a ServiceInstance resource: You can deploy Service Instances belonging to different subaccounts within the same namespace.

## Scope <!--need more explanation on this section; how does the number od subaccounts affect the scope of the module?-->

By default, the scope of services of your Kyma instance is the same as of the subaccount in which you created it. You can extend it by adding more subaccounts. Depending on the number of configured Secrets, the scope can range/refer from one to numerous subaccounts.

## Architecture

The SAP BTP Operator module allows for the initial provisioning and retrieving credentials necessary for the application to use a SAP BTP service. <!--rephrase; add more? delete?-->

![SAP BTP Operator architecture](../assets/BtpOperator_architecture.drawio.svg) <!-- why services.cloud.sap.com?-->

### BTP Manager

BTP Manager is an operator based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator CustomResourceDefinition (CRD)](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml). 
BTP Manager performs the following operations:
* provisions the latest version of the SAP BTP service operator along with its resources, Service Instances, and Service Bindings
* updates the SAP BTP service operator
* deprovisions the SAP BTP service operator

### SAP BTP Service Operator

The SAP BTP service operator allows you to connect SAP BTP services to your cluster and then manage them using Kubernetes-native tools. It is responsible for communicating with SAP Service Manager.

### SAP Service Manager

[SAP Service Manager](https://help.sap.com/docs/service-manager/sap-service-manager/sap-service-manager?locale=en-US) is the central registry for service brokers and platforms in SAP BTP, which allows you to:
- consume platform services in any connected runtime environment
- track the creation and management of Service Instances
- share services and service instances between different environments

SAP Service Manager uses [Open Service Broker API](https://www.openservicebrokerapi.org/) to communicate with service brokers.

### Service Broker

<!-- do we need to explain what a service broker is and how it works? isn't it clear to every user at this stage? -->

## API / Custom Resource Definitions

The `btpoperators.operator.kyma-project.io` Custom Resource Definition (CRD) describes the kind and the format of data that BTP Manager <!--is it BTP Manager?--> uses to configure resources.

## Resource Consumption

To learn more about the resources used by the SAP BTP Operator module, see [Kyma Modules' Sizing](https://help.sap.com/docs/btp/sap-business-technology-platform-internal/kyma-modules-sizing?locale=en-US&state=DRAFT&version=Internal#sap-btp-operator).
