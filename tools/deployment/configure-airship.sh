#!/bin/bash

set -xe

: ${MANIFEST_DIR:="${HOME}/vino-manifests"}
: ${VINO_REPO_URL:="/${HOME}/airship/vino"}

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
