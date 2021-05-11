#!/bin/sh

set -xe

# test will assert that these labels exist on the BMHs
: ${NODE_COPY_LABELS:="airshipit.org/server=s1,airshipit.org/rack=r1"}
# test will assert that these labels exist on control plane BMHs
: ${CONTROL_PLANE_LABELS:="airshipit.org/k8s-role=master"}
# test will assert that these labels exist on worker BMHs
: ${WORKER_LABELS:="airshipit.org/k8s-role=worker"}

echo "Checking control plane baremetal hosts created by ViNO" >&2
controlPlaneCount=$(kubectl get baremetalhosts \
    --context "${KCTL_CONTEXT}" \
    --namespace vino-system \
    --selector "${NODE_COPY_LABELS},${CONTROL_PLANE_LABELS}" \
    --output name | wc -l)

echo "Control plane BMH count ${controlPlaneCount}" >&2
# With this test exactly 1 control plane node must have been created by VINO controller
[ "$controlPlaneCount" -eq "1" ]
echo "Control plane BMH count verified" >&2


#Echo "Checking worker baremetal hosts created by ViNO" >&2
#WorkerCount=$(kubectl get baremetalhosts \
#    --context "${KCTL_CONTEXT}" \
#    --namespace vino-system \
#    --selector "${NODE_COPY_LABELS},${WORKER_LABELS}" \
#    --output name | wc -l)
#
#Echo "Worker BMH count ${workerCount}" >&2
## With this test exactly 4 workers must have been created by VINO controller
#[ "$workerCount" -eq "1" ]
#Echo "Worker BMH count verified" >&2
