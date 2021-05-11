#!/bin/sh

set -xe

TIMEOUT=${TIMEOUT:-3600}

end=$(($(date +%s) + $TIMEOUT))

while true; do
    # TODO (kkalynovskyi) figure out how we can handle multiple BMHs
    if [ "$(kubectl get bmh --context $KCTL_CONTEXT -n vino-system -o jsonpath='{.items[*].status.provisioning.state}')" == "ready" ]
    then
        echo "BMH successfully reached provisioning state ready" 1>&2
        break
    else
        now=$(date +%s)
        if [ $now -gt $end ]; then
		echo "BMH(s) didn't reach provisioning state ready in given timeout ${TIMEOUT}" 1>&2
                exit 1
        fi
        sleep 15
    fi
done

