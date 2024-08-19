# BTP Manager

BTP Manager configures and manages the lifecycle of the SAP BTP service operator.

## Module Lifecycle

BTP Manager is an operator for the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing SAP BTP Operator [CustomResourceDefinition](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) (CRD), which allows you to manage the SAP BTP service operator resource through custom resource (CR).

BTP Manager performs the following operations:

* Provisioning of the SAP BTP service operator
* Updating of the SAP BTP service operator
* Deprovisioning of the SAP BTP service operator and its resources, service instances, and service bindings