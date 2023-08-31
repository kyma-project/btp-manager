#! /bin/bash
cd "$(dirname "$0")"
FILE=creds.json
if [ ! -f "$FILE" ];
then
    echo "Required file: $FILE does not exist."
    exit
fi

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
  clientid: $(jq --raw-output '.clientid | @base64' creds.json)
  clientsecret: $(jq --raw-output '.clientsecret | @base64' creds.json)
  sm_url: $(jq --raw-output '.sm_url | @base64' creds.json)
  tokenurl: $(jq --raw-output '.url | @base64' creds.json)
  cluster_id: dGVzdF9jbHVzdGVyX2lk
EOF

echo "operator-secret.yaml file created."
