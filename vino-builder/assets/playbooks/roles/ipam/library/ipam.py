# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
# implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# generate_baremetal_macs method ripped from
# openstack/tripleo-incubator/scripts/configure-vm

import socket
import struct
from itertools import chain
import netaddr


DOCUMENTATION = '''
---
module: ipam
version_added: "1.0"
short_description: Help with IPAM allocation
description:
   - Generate IPs for instances, ensuring they're unique on every node
'''
# we don't support specifying subnet_bridge or instances yet
def allocate_ips(nodes, physical_primary_ip, physical_node_count=1, subnet_bridge='192.168.0.0/24', subnet_instances='192.168.4.0/22'):
    """Return IP assignments"""
    # calculate some stuff
    vm_instance_count = len(nodes)
    last_octet = physical_primary_ip.split('.')[-1]
    node_index = int(last_octet) % int(physical_node_count)
    bridge_ip = netaddr.IPNetwork(subnet_bridge)[node_index+1]
    # generate an ip for every vm in the entire environment
    ip_buckets=[None] * physical_node_count
    vm_ip_list = list(netaddr.IPNetwork(subnet_instances))
    vm_ip_list.reverse()

    # throw away 0, .1, .2, .3 - assumes we won't exceed .255
    vm_ip_list.pop()
    vm_ip_list.pop()
    vm_ip_list.pop()
    vm_ip_list.pop()

    # now take IPs from this list - enough for all the VMs
    # we need to create and place them into groups
    # one for each physical node
    for physnode in range(0, physical_node_count):
        ip_buckets[physnode] = {}
        ip_list = []
        for vmidx in range(0, vm_instance_count):
            ip_list.append(vm_ip_list.pop().__str__())
        ip_buckets[physnode] = ip_list

    bridge_subnet_netmask = cidr_to_netmask(subnet_bridge)
    return {
        'node_index': node_index,
        'bridge_ip': bridge_ip.__str__(),
        'instance_ips': ip_buckets[node_index],
        'bridge_subnet_netmask': bridge_subnet_netmask,
    }

def cidr_to_netmask(cidr):
    _, net_bits = cidr.split('/')
    host_bits = 32 - int(net_bits)
    netmask = socket.inet_ntoa(struct.pack('!I', (1 << 32) - (1 << host_bits)))
    return netmask

def main():
    module = AnsibleModule(
        argument_spec=dict(
            nodes=dict(required=True, type='list'),
            physical_node_count=dict(required=True, type='int'),
            primary_ipaddress=dict(required=True, type='str'),
            subnet_bridge=dict(required=True, type='str'),
            subnet_instances=dict(required=True, type='str'),
        )
    )
    result = allocate_ips(module.params["nodes"],
                          module.params["primary_ipaddress"],
                          module.params["physical_node_count"],
                          module.params["subnet_bridge"],
                          module.params["subnet_instances"])
    module.exit_json(**result)
# see http://docs.ansible.com/developing_modules.html#common-module-boilerplate
from ansible.module_utils.basic import AnsibleModule  # noqa
if __name__ == '__main__':
    main()
