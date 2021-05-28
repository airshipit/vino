#!/bin/bash

set -xe

: ${MANIFEST_DIR:="${HOME}/vino-manifests"}
: ${VINO_REPO_URL:="/${HOME}/airship/vino"}
AIRSHIPCTL_RELEASE=${AIRSHIPCTL_RELEASE:-"v2.0.0"}

mkdir -p "${MANIFEST_DIR}"

# Workaround for testing against local changes with vino
if [ -d "${VINO_REPO_URL}" ]; then
    VINO_REPO_URL=$(realpath "${VINO_REPO_URL}")
    cp -r "${VINO_REPO_URL}" "${MANIFEST_DIR}/"
fi

if [ ! -f "${HOME}/.airship/config" ]; then
    airshipctl config init
fi

airshipctl config set-manifest default \
    --target-path "${MANIFEST_DIR}" \
    --repo primary \
    --metadata-path config/phases/metadata.yaml \
    --url ${VINO_REPO_URL}

airshipctl config set-manifest default \
    --repo airshipctl \
    --url https://opendev.org/airship/airshipctl.git \
    --branch master

airshipctl document pull -n


# pinning airshipctl to a specific tag
# git checkout is manually done here as a workaround to checkout a specific version of airshipctl.
# `airshipctl document pull -n` does not respect branch/tag
# -n is required for vino while a specific tag is required for airshipctl
cd ${MANIFEST_DIR}/airshipctl
git checkout ${AIRSHIPCTL_RELEASE}
