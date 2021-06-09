#!/bin/bash

set -xe

function create_bridge () {
    if ! sudo brctl show| grep -q "${1}"; then
        sudo brctl addbr "${1}"
        sudo ip link set "${1}" up
        sudo ip addr add ${2} dev "${1}"
    fi;
}

VM_INFRA_BRIDGE=${VM_INFRA_BRIDGE:-"vm-infra"}
VM_INFRA_BRIDGE_IP=${VM_INFRA_BRIDGE_IP:-"192.168.2.1/24"}

VM_PXE_BRIDGE=${VM_PXE_BRIDGE:-"ironic-bridge"}
VM_PXE_BRIDGE_IP=${VM_PXE_BRIDGE_IP:-"172.3.3.1/24"}
PXE_NET="172.3.3.0/24"

export DEBCONF_NONINTERACTIVE_SEEN=true
export DEBIAN_FRONTEND=noninteractive

sudo modprobe dummy

sudo ip link add enp3s7 type dummy
sudo ip addr add dev enp3s7 "192.168.2.1/24"
sudo ip link set up dev enp3s7

sudo -E apt-get update
sudo -E apt-get install -y bridge-utils

echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward

create_bridge ${VM_PXE_BRIDGE} ${VM_PXE_BRIDGE_IP}