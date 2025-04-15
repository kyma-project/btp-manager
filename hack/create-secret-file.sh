#! /bin/bash
cd "$(dirname "$0")"

# Option tells for which application we are making secret
OPTION=${1:-'manager'}
CREDS=${3:-'creds.json'}

if [ ! -f "$CREDS" ]; then
    echo "Required file: $CREDS does not exist."
    exit 1
fi

if [[ $OPTION != 'manager' && $OPTION != 'operator' ]]; then 
  echo "unsupported option passed $OPTION"
  exit 1
fi  

if [ "$OPTION" == "manager" ]; then
echo 'secret for BTP Manager will be created' 
cat <<EOF > operator-secret.yaml
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
echo ''secret: operator-secret.yaml for BTP Manager created' ' 
exit 0
fi

if [ "$OPTION" == "operator" ]; then
echo 'secret with BTP access credentials for SAP BTP Service Operator will be created' 
cat <<EOF > btp-access-credentials-secret.yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: $SECRETNAME
  namespace: kyma-system
data:
  clientid: $(jq --raw-output '.clientid | @base64' ${CREDS})
  clientsecret: $(jq --raw-output '.clientsecret | @base64' ${CREDS})
  sm_url: $(jq --raw-output '.sm_url | @base64' ${CREDS})
  tokenurl: $(jq --raw-output '.url | @base64' ${CREDS})
  tokenurlsuffix: L29hdXRoL3Rva2Vu
EOF
echo 'secret: btp-access-credentials-secret.yaml with BTP access credentials for SAP BTP Service Operator created'
exit 0
fi

echo 'Unsupported case'
exit 1 