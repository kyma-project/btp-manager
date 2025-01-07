# sap-btp-operator Helm chart

## Overview

This is a custom version of the sap-btp-operator Helm chart.

The upstream version of the sap-btp-operator Helm chart has a dependency on the Jetstack cert-manager. This custom version makes [jetstack/cert-manager](https://github.com/jetstack/cert-manager) optional and adds the possibility to use a custom caBundle or [gardener/cert-management](https://github.com/gardener/cert-management).

## Prerequisites

* Kubernetes 1.16+
* Helm 3+

## Install chart

<details>
<summary>With fixed caBundle</summary>

helm install sap-btp-operator . \
    --atomic \
    --create-namespace \
    --namespace=sap-btp-operator \
    --set manager.secret.clientid="<fill in>" \
    --set manager.secret.clientsecret="<fill in>" \
    --set manager.secret.url="<fill in>" \
    --set manager.secret.tokenurl="<fill in>" \
    --set cluster.id="<fill in>"

</details>

<details>
<summary>With custom caBundle</summary>

helm install sap-btp-operator . \
    --atomic \
    --create-namespace \
    --namespace=sap-btp-operator \
    --set manager.secret.clientid="<fill in>" \
    --set manager.secret.clientsecret="<fill in>" \
    --set manager.secret.url="<fill in>" \
    --set manager.secret.tokenurl="<fill in>" \
    --set manager.certificates.selfSigned.caBundle="${CABUNDLE}" \
    --set manager.certificates.selfSigned.crt="${SERVERCRT}" \
    --set manager.certificates.selfSigned.key="${SERVERKEY}" \
    --set cluster.id="<fill in>"

</details>

<details>
<summary>With jetstack/cert-manager</summary>

helm install sap-btp-operator . \
    --atomic \
    --create-namespace \
    --namespace=sap-btp-operator \
    --set manager.secret.clientid="<fill in>" \
    --set manager.secret.clientsecret="<fill in>" \
    --set manager.secret.url="<fill in>" \
    --set manager.secret.tokenurl="<fill in>" \
    --set manager.certificates.certManager=true \
    --set cluster.id="<fill in>"
  
  </details>

<details>
<summary>With gardener/cert-management</summary>

helm template sap-btp-operator . \
    --atomic \
    --create-namespace \
    --namespace=sap-btp-operator \
    --set manager.secret.clientid="<fill in>" \
    --set manager.secret.clientsecret="<fill in>" \
    --set manager.secret.url="<fill in>" \
    --set manager.secret.tokenurl="<fill in>" \
    --set manager.certificates.certManagement.caBundle="${CABUNDLE}" \
    --set manager.certificates.certManagement.crt=${CACRT} \
    --set manager.certificates.certManagement.key=${CAKEY} \
    --set cluster.id="<fill in>"

</details>


## Overrides
While rendering Kubernetes resource files by Helm, the following [values overrides](https://github.com/kyma-project/btp-manager/blob/main/module-chart/overrides.yaml) are applied.

## Publish a new version of the chart
1.  Download the original chart from the Helm repository  
   
     i. Configure the Helm repository
    ```
     helm repo add sap-btp-operator https://sap.github.io/sap-btp-service-operator
    ```  
    ii. Pull the chart

    ```
    helm pull sap-btp-operator/sap-btp-operator
    ```
    > **NOTE:** You can specify the version if needed
    >```
    >helm pull sap-btp-operator/sap-btp-operator --version v0.2.0
    >```

    iii. Unpack the downloaded tar and apply necessary changes.

1. Create a package
   ```
   helm package chart 
   ```
1. Release on GitHub  
Create a GitHub release and upload the generated Helm chart (tgz).
