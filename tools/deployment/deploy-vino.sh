#!/bin/bash

set -xe
sudo snap install kustomize && sudo snap install go --classic
make docker-build
make deploy
kubectl get po -A
#Wait for vino controller manager Pod.
kubectl wait -n vino-system pod  -l control-plane=controller-manager --for=condition=ready --timeout=240s