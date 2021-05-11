#!/bin/sh

set -xe

# If NODE_NAME is not set, all nodes will be labeled
: ${NODE_NAME:="--all"}

# Node label example: NODE_LABELS="airshipit.org/rack=r1 airshipit.org/server=s1"
: ${NODE_LABELS:=""}

echo "Labeling node(s) ${NODE_NAME} with labels ${NODE_LABELS}" >&2
kubectl label node \
  --context $KCTL_CONTEXT \
  --overwrite \
  ${NODE_NAME} ${NODE_LABELS} >&2
