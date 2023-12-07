# Install and Uninstall BTP Manager

## Install BTP Manager Locally

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

Use the following commands to run the BTP Manager controller from your host. Both `make` commands refer to [Makefile](../../Makefile).

```sh
make install
make run
```

## Install BTP Manager in Your Cluster

There are three ways to install BTP Manager in your cluster:

<!-- tabs:start -->

#### With kubectl and `btp-manager.yaml` (recommended)  

You need the following prerequisites:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

Use the following command to download and install BTP Manager from Kubernetes resources in your cluster:

```shell
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```

Use the following command to uninstall BTP Manager from your cluster:

```shell
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
kubectl delete -f deployments/prerequisites.yaml
```

#### With Helm and `template.yaml`  

You need the following prerequisites:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io))
- [Helm](https://github.com/helm/helm#install)
- [skopeo](https://github.com/containers/skopeo) 
- [jq](https://github.com/stedolan/jq) 
- [yq](https://github.com/mikefarah/yq) 

To install BTP Manager using a template file (the output of the [kyma alpha create module](https://github.com/kyma-project/cli/blob/main/docs/gen-docs/kyma_alpha_create_module.md) command) in your cluster, use the following command:

```shell
./hack/run_template.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml
```

Use the following command to uninstall BTP Manager from your cluster:
```shell
helm uninstall btp-manager -n kyma-system
```

#### With Lifecycle Manager  

> **NOTE:** This is an experimental way of installing BTP Manager in your cluster.

You need the following prerequisites:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [k3d](https://k3d.io)

Here is the Quick-Start - Install script.

Use the following command to run the BTP Manager with Lifecycle Manager: 

```shell
./hack/run_lifecycle_manager.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml
```

It results in:
- downloading and using Kyma CLI to provision the k3d cluster
- deploying Lifecycle Manager
- applying the BTP Manager `template.yaml` provided by the user
- enabling the SAP BTP Operator module
- displaying the BTP Manager and SAP BTP Operator status

To delete your k3d cluster, use the following command:

```shell
k3d cluster delete kyma
```

<!-- tabs:end -->
