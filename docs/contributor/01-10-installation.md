# Install and uninstall BTP Manager

## Install BTP Manager locally

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

Use the following commands to run the BTP Manager controller from your host. Both `make` commands refer to [Makefile](../../Makefile).

```sh
make install
make run
```

## Install BTP Manager in your cluster

There are three ways to install BTP Manager in your cluster:

<details>
<summary>With kubectl and <code>btp-manager.yaml</code> (recommended)</summary>
<br>

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

Use the following command to download and install BTP Manager from Kubernetes resources in your cluster.

```shell
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```

Use the following command to uninstall BTP Manager from your cluster.

```shell
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
kubectl delete -f deployments/prerequisites.yaml
```
</details>

<details>
<summary>With Helm and <code>template.yaml</code></summary>
<br>

### Prerequisites

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

Use the following command to uninstall BTP Manager from your cluster.
```shell
helm uninstall btp-manager -n kyma-system
```

</details>

<details>
<summary>With Lifecycle Manager</summary>
<br>

> **NOTE:** This is an experimental way of installing BTP Manager in your cluster.

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [k3d](https://k3d.io)

### Quick-Start - Install script

Use the following command to run the BTP Manager with Lifecycle Manager. 

```shell
./hack/run_lifecycle_manager.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml
```

It results in:
- downloading and using Kyma CLI to provision the k3d cluster
- deploying Lifecycle Manager
- applying the BTP Manager `template.yaml` provided by the user
- enabling the SAP BTP Operator module
- displaying the BTP Manager and SAP BTP Operator status

### Delete k3d cluster

```shell
k3d cluster delete kyma
```

</details> 
