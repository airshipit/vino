#!/bin/sh

set -xe

TIMEOUT=${TIMEOUT:-600}

end=$(($(date +%s) + $TIMEOUT))

timeout 180 kubectl delete vino --all --context $KCTL_CONTEXT >&2
node_name=$(kubectl --context $KCTL_CONTEXT get node -o name)
while true; do
    annotation=$(kubectl --context $KCTL_CONTEXT get $node_name -o=jsonpath="{.metadata.annotations.airshipit\.org/vino\.network-values}")
    if [ "${annotation}" == "" ]
    then
        echo "Succesfuly remove annotation from a node" >&2
        break
    else
        now=$(date +%s)
        if [ $now -gt $end ]; then
		      echo "Failed to removed annotation from node ${node_name} after deleting vino CR, exiting" >&2
          exit 1
        fi
        sleep 15
    fi
done
