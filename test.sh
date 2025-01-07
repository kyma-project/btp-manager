#!/bin/bash

# Function to check if the last command was successful
check_command_success() {
    if [ $? -ne 0 ]; then
        echo "An error occurred during the last operation. Exiting."
        exit 1
    fi
}

# Check if cluster "test" exists and delete it if found
existingCluster=$(k3d cluster list | grep -w "test")
if [ -n "$existingCluster" ]; then
    echo "Deleting existing cluster 'test'..."
    k3d cluster delete test
    check_command_success
fi

# Create new cluster
echo "Creating new cluster 'test'..."
k3d cluster create test
check_command_success

# Wait a moment for the cluster to be ready
echo "Waiting for the cluster to be ready..."
sleep 10

# Apply Kubernetes configurations
echo "Applying Kubernetes configurations..."

# Check if the namespace already exists, create it if not
kubectl get namespace kyma-system >/dev/null 2>&1
if [ $? -ne 0 ]; then
    kubectl create namespace kyma-system
    check_command_success
else
    echo "Namespace 'kyma-system' already exists. Skipping creation."
fi

# Apply the manifests
for manifest in test-secret.yaml btp-manager.yaml btp-operator-default-cr.yaml; do
    echo "Applying $manifest..."
    kubectl apply -f $manifest
    check_command_success
done

echo "Setup completed successfully!"