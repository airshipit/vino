#!/bin/sh

set -xe

# Name of the daemonset
: ${DAEMONSET_NAME:="default-vino-test-cr"}
# Namespace of the daemonset
: ${DAEMONSET_NAMESPACE:="vino-system"}
# Maximum retries
: ${MAX_RETRY:="30"}
# How long to wait between retries in seconds
: ${RETRY_INTERVAL_SECONDS:="2"}

echo "Verifying that daemonset ${DAEMONSET_NAME} created in namespace ${vino-system} exists" >&2
count=0
until kubectl --context "${KCTL_CONTEXT}" -n "${DAEMONSET_NAMESPACE}" get ds "${DAEMONSET_NAME}" >&2; do
  count=$((count + 1))
  if [ "${count}" -eq "${MAX_RETRY}" ]; then
    echo 'Timed out waiting for daemonset to exist' >&2
    exit 1
  fi
  echo "Retrying to get daemonset attempt ${count}/${MAX_RETRY}" >&2
  sleep "${RETRY_INTERVAL_SECONDS}"
done

echo "Succesfuly verified that daemonset ${DAEMONSET_NAMESPACE}/${DAEMONSET_NAME} exists" >&2
