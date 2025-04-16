#! /bin/bash

if [ "$1" == "operator" ]; then
  SECRETNAME=$2
  [[ -z $SECRETNAME ]] && echo "for option 'operator' secret name parameter is required" && exit 1
  CREDS=${3:-"creds.json"}
  [[ ! -f "$CREDS" ]] && echo "required file $CREDS not found" && exit 1
  echo 'secret with BTP access credentials for SAP BTP Service Operator will be created'
  cat <<EOF >btp-access-credentials-secret.yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: ${SECRETNAME}
  namespace: kyma-system
data:
  clientid: $(jq --raw-output '.clientid | @base64' ${CREDS})
  clientsecret: $(jq --raw-output '.clientsecret | @base64' ${CREDS})
  sm_url: $(jq --raw-output '.sm_url | @base64' ${CREDS})
  tokenurl: $(jq --raw-output '.url | @base64' ${CREDS})
  tokenurlsuffix: L29hdXRoL3Rva2Vu
EOF
    echo 'secret: btp-access-credentials-secret.yaml with BTP access credentials for SAP BTP Service Operator created'
else
  CREDS=${1:-"creds.json"}
  [[ ! -f "$CREDS" ]] && echo "required file $CREDS not found" && exit 1
  echo 'secret for BTP Manager will be created'
  cat <<EOF >operator-secret.yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: sap-btp-manager
  namespace: kyma-system
  labels:
    app.kubernetes.io/managed-by: kcp-kyma-environment-broker
data:
  clientid: $(jq --raw-output '.clientid | @base64' ${CREDS})
  clientsecret: $(jq --raw-output '.clientsecret | @base64' ${CREDS})
  sm_url: $(jq --raw-output '.sm_url | @base64' ${CREDS})
  tokenurl: $(jq --raw-output '.url | @base64' ${CREDS})
  cluster_id: dGVzdF9jbHVzdGVyX2lk
EOF
  echo 'secret: operator-secret.yaml for BTP Manager created'
fi
