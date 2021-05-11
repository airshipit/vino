#!/bin/bash

set -xe

export AIRSHIPCTL_VERSION=${AIRSHIPCTL_VERSION:-"2.0.0"}
airship_release_url="https://github.com/airshipit/airshipctl/releases/download/v${AIRSHIPCTL_VERSION}/airshipctl_${AIRSHIPCTL_VERSION}_linux_amd64.tar.gz"

wget -q -c "${airship_release_url}" -O - | sudo tar -xz -C /usr/local/bin/
