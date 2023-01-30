# sap-btp-operator Helm chart

## Overview

This is a custom version of the sap-btp-operator Helm chart.

The upstream version of the sap-btp-operator Helm chart has a dependency on the jetstack cert-manager. This custom version makes [jetstack/cert-manager](https://github.com/jetstack/cert-manager) optional and adds the possibility to use a custom caBundle or [gardener/cert-management](https://github.com/gardener/cert-management).

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


## Change the original chart to the sap-btp-operator Helm chart 
1. Disable Istio.

Add the annotation to the deployment:
```
sidecar.istio.io/inject: "false"
```

2. Move Secrets into `webhook.yml` and define certificates:
```yaml
{{- $cn := printf "sap-btp-operator-webhook-service"  }}
{{- $ca := genCA (printf "%s-%s" $cn "ca") 3650 }}
{{- $altName1 := printf "%s.%s" $cn .Release.Namespace }}
{{- $altName2 := printf "%s.%s.svc" $cn .Release.Namespace }}
{{- $cert := genSignedCert $cn nil (list $altName1 $altName2) 3650 $ca }}
{{- if not .Values.manager.certificates }}
apiVersion: v1
kind: Secret
metadata:
  name: webhook-server-cert
  namespace: {{.Release.Namespace}}
type: kubernetes.io/tls
data:
  tls.crt: {{ b64enc $cert.Cert }}
  tls.key: {{ b64enc $cert.Key }}
---
apiVersion: v1
kind: Secret
metadata:
  name: sap-btp-service-operator-tls
  namespace: {{ .Release.Namespace }}
type: kubernetes.io/tls
data:
  tls.crt: {{ b64enc $cert.Cert }}
  tls.key: {{ b64enc $cert.Key }}
---
{{- end}}
```
3. Add the `caBundle` definition in both webhooks:
```
{{- if not .Values.manager.certificates }}
caBundle: {{ b64enc $ca.Cert }}
{{- end }}
```

4. Add sap-btp-operator labels

The deployment and service must contain btp-operator specific labels, such as deployment spec, deployment template, and the service labels selector:
```yaml
app.kubernetes.io/instance: sap-btp-operator
app.kubernetes.io/name: sap-btp-operator
```

## Publish a new version of the chart
1.  Download the original chart from the Helm repository.  
   
     i. Configure the Helm repository:
    ```
     helm repo add sap-btp-operator https://sap.github.io/sap-btp-service-operator
    ```  
    ii. Pull the chart

    ```
    helm pull sap-btp-operator/sap-btp-operator
    ```
    > **NOTE:** You can specify the version if needed:
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
