#!/bin/bash
set -xe
curl -s -L https://opendev.org/airship/charts/raw/branch/master/tools/gate/deploy-k8s.sh | bash
sudo snap install kustomize && sudo snap install go --classic
#Wait for all pods to be ready before starting Vino Image build.
kubectl wait --for=condition=Ready pods --all -A --timeout=180s
make docker-build
kustomize build config/default | kubectl apply -f -
kubectl get po -A
#Wait for vino controller manager Pod.
kubectl wait -n vino-system pod  -l control-plane=controller-manager --for=condition=ready --timeout=240s