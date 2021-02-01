#!/bin/bash

set -xe
sudo snap install kustomize && sudo snap install go --classic
make docker-build-controller
make deploy
kubectl get po -A
#Wait for vino controller manager Pod.
count=0
until [[ $(kubectl -n vino-system get pod -l control-plane=controller-manager 2>/dev/null) ]]; do
  count=$((count + 1))
  if [[ ${count} -eq "120" ]]; then
    echo ' Timed out waiting for vino controller manager pod to exist'
    return 1
  fi
  sleep 2
done
kubectl wait -n vino-system pod  -l control-plane=controller-manager --for=condition=ready --timeout=240s