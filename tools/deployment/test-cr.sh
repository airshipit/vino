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
until [[ $(kubectl get ds vino-test-cr 2>/dev/null) ]]; do
  count=$((count + 1))
  if [[ ${count} -eq "30" ]]; then
    echo ' Timed out waiting for vino builder daemonset to exist'
    vinoDebugInfo
    return 1
  fi
  sleep 2
done
if ! kubectl rollout status ds vino-test-cr --timeout=10s; then
    vinoDebugInfo
fi
