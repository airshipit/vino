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
if ! kubectl wait --for=condition=Ready vino vino-test-cr --timeout=180s; then
    vinoDebugInfo
fi

# no need to collect logs on fail, since they are already collected before
if ! kubectl wait --for=condition=Ready pods -l 'vino-role=vino-builder' --timeout=5s; then
    vinoDebugInfo
fi
