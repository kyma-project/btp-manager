#!/bin/bash

modularizationPath="/Users/lj/Go/src/github.com/kyma-project/modularization"

applyKymaModule () {
    cd "$modularizationPath/$1" && 
    make install &&
    cd .. && 
    make module-build && 
    make module-image && 
    make module-template-push
}

if [ "$1" = "rc" ]; then

        clusters=$(k3d cluster list)

        if [[ $clusters = *"op-"* ]]; then
            k3d cluster delete op-skr &&
            k3d cluster delete op-kcp
        fi

        k3d cluster create op-skr --registry-create op-skr-registry.localhost &&
        k3d cluster create op-kcp --registry-create op-kcp-registry.localhost && 

        kubectl config use k3d-op-skr &&
        kubectl create clusterrolebinding serviceaccounts-cluster-admin --clusterrole=cluster-admin --group=system:serviceaccounts &&
        kubectl config use k3d-op-kcp &&

        skrreg=$(docker port op-skr-registry.localhost)
        skrregport=${skrreg#*:}
        kcpreg=$(docker port op-kcp-registry.localhost)
        kcpregport=${kcpreg#*:}

        sed -i .orig "6s/.*/MODULE_REGISTRY_PORT ?= $kcpregport/" "$modularizationPath/btp-manager/Makefile" &&
        sed -i .orig "16s/.*/IMG_REGISTRY_PORT ?= $skrregport/" "$modularizationPath/btp-manager/Makefile" &&

        sed -i .orig "6s/.*/MODULE_REGISTRY_PORT ?= $kcpregport/" "$modularizationPath/lifecycle-manager/samples/template-operator/Makefile" &&
        sed -i .orig "16s/.*/IMG_REGISTRY_PORT ?= $skrregport/" "$modularizationPath/lifecycle-manager/samples/template-operator/Makefile" &&

        kubectl config use k3d-op-kcp &&

        kubectl create ns kyma-system &&

        export MODULE_REGISTRY_PORT=$(docker port op-kcp-registry.localhost 5000/tcp | cut -d ":" -f2) &&
        export IMG_REGISTRY_PORT=$(docker port op-skr-registry.localhost 5000/tcp | cut -d ":" -f2) &&

        cd "$modularizationPath/lifecycle-manager/operator" && 
        make install &&

        osascript -e 'tell app "Terminal"
            do script "cd '${modularizationPath}'/lifecycle-manager/operator && make run"
        end tell' &&

        cd "$modularizationPath/lifecycle-manager/operator" &&    
        sh config/samples/secret/k3d-secret-gen.sh
fi

kubectl config use k3d-op-kcp &&

applyKymaModule "lifecycle-manager/samples/template-operator/operator" &&

applyKymaModule "btp-manager/operator" &&

cd "$modularizationPath/btp-manager/" && 
sh hack/gen-kyma.sh  && 
kubectl apply -f kyma.yaml
