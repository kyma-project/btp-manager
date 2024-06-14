# BTP Manager User Interface

> [!WARNING]
> This feature is in the experimental stage and is not yet available in the main branch or official releases.
> Use the latest development image to test the UI: europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-720

## Prerequisites
For clusters different from Kyma (e.g. k3d), you need to install the [prerequisites](../../deployments/prerequisites.yaml).
```shell
kubectl apply -f deployments/prerequisites.yaml
```


## Run BTP Manager with UI
Follow steps below to run BTP Manager with UI:
1. Connect `kubectl` to your cluster by setting the **KUBECONFIG** environment variable.
    ```shell
    export KUBECONFIG=<path-to-kubeconfig>
    ```
2. Make sure the `btp-operator` module is disabled and there are no existing BtpOperator custom resources (CRs) and deployments of BTP Manager and the SAP BTP service operator.
    ```shell
    kubectl get btpoperators -A
    kubectl get deployment -n kyma-system btp-manager-controller-manager
    kubectl get deployment -n kyma-system sap-btp-operator-controller-manager
    ```
3. Clone the `btp-manager` repository and checkout to the `sm-integration` branch.
    ```shell
    git clone https://github.com/kyma-project/btp-manager.git
    git checkout sm-integration
    ```
4. Set **IMG** environment variable to the image of BTP Manager with UI.
    ```shell
    export IMG=europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-720
    ```
5. Run `deploy` makefile rule to deploy BTP Manager with UI.
    ```shell
    make deploy
    ```
6. Check if BTP Manager deployment is running.
    ```shell
    kubectl get deployment -n kyma-system btp-manager-controller-manager
    ```
    If you encounter the following error during Pod creation due to Warden's admission webhook:
    ```
    Error creating: admission webhook "validation.webhook.warden.kyma-project.io" denied the request: Pod images europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-720 validation failed
    ```
    you must scale the BTP Manager deployment to 0 replicas, delete the webhook, and the scale the deployment back to 1 replica.
    ```shell
    kubectl scale deployment -n kyma-system btp-manager-controller-manager --replicas=0
    kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io validation.webhook.warden.kyma-project.io
    kubectl scale deployment -n kyma-system btp-manager-controller-manager --replicas=1
    ```
7. Apply BtpOperator CR to create the Secret with credentials to access Service Manager.
    ```shell
    kubectl apply -n kyma-system -f examples/btp-operator.yaml
    ```
8. Port-forward to BTP Manager deployment.
    ```shell
    kubectl port-forward -n kyma-system deployment/btp-manager-controller-manager 8080:8080
    ```
9. Access the UI by opening `localhost:8080` in your browser.

### Cleanup
After testing UI, you can delete BtpOperator custom resource BTP Manager deployment.
1. Delete BtpOperator custom resource after testing.
    ```shell
    kubectl delete -n kyma-system btpoperator btpoperator
    ```
2. Delete BTP Manager deployment after testing by running `undeploy` makefile rule.
    ```shell
    make undeploy
    ```