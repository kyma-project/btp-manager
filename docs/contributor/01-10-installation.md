# Install and Uninstall BTP Manager

## Install BTP Manager Locally

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

To run the BTP Manager controller from your host, use the following commands. Both `make` commands refer to [Makefile](../../Makefile).

```sh
make install
make run
```

## Install BTP Manager in Your Cluster

Use the following command to download and install BTP Manager from the Kubernetes resources in your cluster:

```shell
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```

Use the following command to uninstall BTP Manager from your cluster:

```shell
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
kubectl delete -f deployments/prerequisites.yaml
```
