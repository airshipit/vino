#!/bin/bash

set -xe

# TODO (kkalynovskyi) remove this function when zuul is able to gather debug info by itself
function vinoDebugInfo () {
    kubectl get po -A
    kubectl get ds -A
    local pod_name
    pod_name="$(kubectl get pod -n vino-system -l control-plane=controller-manager -o name)"
    kubectl logs -c manager ${pod_name} -n vino-system
    exit 1
}

kubectl apply -f config/samples/vino_cr.yaml
kubectl apply -f config/samples/ippool.yaml
kubectl apply -f config/samples/network-template-secret.yaml

# Remove logs collection from here, when we will have zuul collect logs job
until [[ $(kubectl get vino vino-test-cr 2>/dev/null) ]]; do
  count=$((count + 1))
  if [[ ${count} -eq "30" ]]; then
    echo ' Timed out waiting for vino test cr to exist'
    vinoDebugInfo
    return 1
  fi
  sleep 2
done
if ! kubectl wait --for=condition=Ready vino vino-test-cr --timeout=180s; then
    vinoDebugInfo
fi

# no need to collect logs on fail, since they are already collected before
until [[ $(kubectl -n vino-system get ds default-vino-test-cr 2>/dev/null) ]]; do
  count=$((count + 1))
  if [[ ${count} -eq "30" ]]; then
    echo ' Timed out waiting for vino builder daemonset to exist'
    vinoDebugInfo
    return 1
  fi
  sleep 2
done
if ! kubectl -n vino-system rollout status ds default-vino-test-cr --timeout=10s; then
    vinoDebugInfo
fi

bmhCount=$(kubectl get baremetalhosts -n vino-system -o name | wc -l)

# with this setup set up, exactly 3 BMHs must have been created by VINO controller

[[ "$bmhCount" -eq "3" ]]

kubectl get secret -o yaml -n vino-system default-vino-test-cr-worker
