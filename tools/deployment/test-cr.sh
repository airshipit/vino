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

server_label="airshipit.org/server=s1"
rack_label="airshipit.org/rack=r1"
master_copy_label="airshipit.org/k8s-role=master"
worker_copy_label="airshipit.org/k8s-role=worker"

# Label all nodes with the same rack/label. We are ok with this for this simple test.
kubectl label node --overwrite=true --all $server_label $rack_label

kubectl apply -f config/samples/vino_cr_4_workers_1_cp.yaml
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
if ! kubectl wait --for=condition=Ready vino vino-test-cr --timeout=600s; then
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

masterCount=$(kubectl get baremetalhosts -n vino-system -l "$server_label,$server_label,$master_copy_label" -o name | wc -l)
# with this setup set up, exactly 1 master must have been created by VINO controller
[[ "$masterCount" -eq "1" ]]

workerCount=$(kubectl get baremetalhosts -n vino-system -l "$server_label,$server_label,$worker_copy_label" -o name | wc -l)
# with this setup set up, exactly 4 workers must have been created by VINO controller
[[ "$workerCount" -eq "4" ]]

kubectl get baremetalhosts -n vino-system --show-labels=true

kubectl get -o yaml -n vino-system \
   $(kubectl get secret -o name -n vino-system | grep network-data)
