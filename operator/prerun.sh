k3d  cluster delete dem &&
k3d  cluster create dem &&
kubectl config use k3d-dem &&

cd /Users/lj/Go/src/github.com/kyma-project/modularization/btp-manager/operator &&
make install &&
cd /Users/lj/Go/src/github.com/kyma-project/modularization/btp-manager/ &&
kubectl apply -f default.yaml
