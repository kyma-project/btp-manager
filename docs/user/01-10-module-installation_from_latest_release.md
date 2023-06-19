---
Install BPT Operator module from latest release
---

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster, or [k3d](https://k3d.io) for local installation
  
To enable the BTP Operator module from the latest release, you must install BTP Manager and SAP BTP Service Operator. Follow these steps to do that:

1. To install BTP Manager, use the following command:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/rendered.yaml
    ```
    > **TIP:** Use the same command to upgrade the module to the latest version.

<br>

 2. To install SAP BTP Service Operator, apply the sample BtpOperator CR:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/tree/main/config/samples
    ```


For more details on other installation options, read the [Install and uninstall BTP Manager](../contributor/01-10-installation.md) document.
