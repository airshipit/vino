<h1>Vino API reference</h1>
<p>Packages:</p>
<ul class="simple">
<li>
<a href="#airship.airshipit.org%2fv1">airship.airshipit.org/v1</a>
</li>
</ul>
<h2 id="airship.airshipit.org/v1">airship.airshipit.org/v1</h2>
<p>Package v1 contains API Schema definitions for the airship v1 API group</p>
Resource Types:
<ul class="simple"></ul>
<h3 id="airship.airshipit.org/v1.AllocatedIP">AllocatedIP
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.IPPoolSpec">IPPoolSpec</a>)
</p>
<p>AllocatedIP Allocates an IP and MAC address to an entity</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>mac</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allocatedTo</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.BMCCredentials">BMCCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>BMCCredentials contain credentials that will be used to create BMH nodes
sushy tools will use these credentials as well, to set up authentication</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>username</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>password</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.Builder">Builder
</h3>
<p>TODO (kkalynovskyi) create an API object for this, and refactor vino-builder to read it from kubernetes.</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>gwIPBridge</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHost</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHostPort</code><br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>networks</code><br>
<em>
<a href="#airship.airshipit.org/v1.BuilderNetwork">
[]BuilderNetwork
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>configuration</code><br>
<em>
<a href="#airship.airshipit.org/v1.CPUConfiguration">
CPUConfiguration
</a>
</em>
</td>
<td>
<p>(TODO) change json tag to cpuConfiguration when vino-builder has these chanages as well</p>
</td>
</tr>
<tr>
<td>
<code>domains</code><br>
<em>
<a href="#airship.airshipit.org/v1.BuilderDomain">
[]BuilderDomain
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>nodeCount</code><br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.BuilderDomain">BuilderDomain
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Builder">Builder</a>)
</p>
<p>BuilderDomain represents a VINO libvirt domain</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>role</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bootMACAddress</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>enableVNC</code><br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vncPassword</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>interfaces</code><br>
<em>
<a href="#airship.airshipit.org/v1.BuilderNetworkInterface">
[]BuilderNetworkInterface
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.BuilderNetwork">BuilderNetwork
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Builder">Builder</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bridgeIP</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bridgeMAC</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>range</code><br>
<em>
<a href="#airship.airshipit.org/v1.Range">
Range
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Network</code><br>
<em>
<a href="#airship.airshipit.org/v1.Network">
Network
</a>
</em>
</td>
<td>
<p>
(Members of <code>Network</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.BuilderNetworkInterface">BuilderNetworkInterface
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.BuilderDomain">BuilderDomain</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ipAddress</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>macAddress</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>NetworkInterface</code><br>
<em>
<a href="#airship.airshipit.org/v1.NetworkInterface">
NetworkInterface
</a>
</em>
</td>
<td>
<p>
(Members of <code>NetworkInterface</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.CPUConfiguration">CPUConfiguration
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Builder">Builder</a>, 
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>CPUConfiguration CPU node configuration</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>cpuExclude</code><br>
<em>
string
</em>
</td>
<td>
<p>Exclude CPU example 0-4,54-60</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.DaemonSetOptions">DaemonSetOptions
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>DaemonSetOptions be used to spawn vino-builder, libvirt, sushy an</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespacedName</code><br>
<em>
<a href="#airship.airshipit.org/v1.NamespacedName">
NamespacedName
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>libvirtImage</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sushyImage</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vinoBuilderImage</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>nodeAnnotatorImage</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.DiskDrivesTemplate">DiskDrivesTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.NodeSet">NodeSet</a>)
</p>
<p>DiskDrivesTemplate defines disks on the VM</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>path</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>options</code><br>
<em>
<a href="#airship.airshipit.org/v1.DiskOptions">
DiskOptions
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.DiskOptions">DiskOptions
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.DiskDrivesTemplate">DiskDrivesTemplate</a>)
</p>
<p>DiskOptions disk options</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>sizeGb</code><br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sparse</code><br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.IPPool">IPPool
</h3>
<p>IPPool is the Schema for the ippools API</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br>
<em>
<a href="#airship.airshipit.org/v1.IPPoolSpec">
IPPoolSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>subnet</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ranges</code><br>
<em>
<a href="#airship.airshipit.org/v1.Range">
[]Range
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allocatedIPs</code><br>
<em>
<a href="#airship.airshipit.org/v1.AllocatedIP">
[]AllocatedIP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>macPrefix</code><br>
<em>
string
</em>
</td>
<td>
<p>MACPrefix defines the MAC prefix to use for VM mac addresses</p>
</td>
</tr>
<tr>
<td>
<code>nextMAC</code><br>
<em>
string
</em>
</td>
<td>
<p>NextMAC indicates the next MAC address (in sequence) that
will be provisioned to a VM in this Subnet</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br>
<em>
<a href="#airship.airshipit.org/v1.IPPoolStatus">
IPPoolStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.IPPoolSpec">IPPoolSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.IPPool">IPPool</a>)
</p>
<p>IPPoolSpec tracks allocation ranges and statuses within a specific
subnet IPv4 or IPv6 subnet.  It has a set of ranges of IPs
within the subnet from which IPs can be allocated by IPAM,
and a set of IPs that are currently allocated already.</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>subnet</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ranges</code><br>
<em>
<a href="#airship.airshipit.org/v1.Range">
[]Range
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allocatedIPs</code><br>
<em>
<a href="#airship.airshipit.org/v1.AllocatedIP">
[]AllocatedIP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>macPrefix</code><br>
<em>
string
</em>
</td>
<td>
<p>MACPrefix defines the MAC prefix to use for VM mac addresses</p>
</td>
</tr>
<tr>
<td>
<code>nextMAC</code><br>
<em>
string
</em>
</td>
<td>
<p>NextMAC indicates the next MAC address (in sequence) that
will be provisioned to a VM in this Subnet</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.IPPoolStatus">IPPoolStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.IPPool">IPPool</a>)
</p>
<p>IPPoolStatus defines the observed state of IPPool</p>
<h3 id="airship.airshipit.org/v1.NamespacedName">NamespacedName
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.DaemonSetOptions">DaemonSetOptions</a>, 
<a href="#airship.airshipit.org/v1.NodeSet">NodeSet</a>)
</p>
<p>NamespacedName to be used to spawn VMs</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.Network">Network
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.BuilderNetwork">BuilderNetwork</a>, 
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>Network defines libvirt networks</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Network Parameter defined</p>
</td>
</tr>
<tr>
<td>
<code>subnet</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>instanceSubnet</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allocationStart</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>allocationStop</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>dns_servers</code><br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>routes</code><br>
<em>
<a href="#airship.airshipit.org/v1.VMRoutes">
[]VMRoutes
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>macPrefix</code><br>
<em>
string
</em>
</td>
<td>
<p>MACPrefix defines the zero-padded MAC prefix to use for
VM mac addresses, and is the first address that will be
allocated sequentially to VMs in this network.
If omitted, a default private MAC prefix will be used.
The prefix should be specified in full MAC notation, e.g.
06:42:42:00:00:00</p>
</td>
</tr>
<tr>
<td>
<code>physicalInterface</code><br>
<em>
string
</em>
</td>
<td>
<p>PhysicalInterface identifies interface into which to plug in libvirt network</p>
</td>
</tr>
<tr>
<td>
<code>libvirtTemplate</code><br>
<em>
string
</em>
</td>
<td>
<p>LibvirtTemplate identifies which libvirt template to be used to create a network</p>
</td>
</tr>
<tr>
<td>
<code>bridgeName</code><br>
<em>
string
</em>
</td>
<td>
<p>BridgeName is the name of the bridge to be created as libvirt network.
works if AllocateNodeIP is sepcified</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.NetworkInterface">NetworkInterface
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.BuilderNetworkInterface">BuilderNetworkInterface</a>, 
<a href="#airship.airshipit.org/v1.NodeSet">NodeSet</a>)
</p>
<p>NetworkInterface define interface on the VM</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Define parameter for network interfaces</p>
</td>
</tr>
<tr>
<td>
<code>type</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>network</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>mtu</code><br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>options</code><br>
<em>
map[string]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.NodeSelector">NodeSelector
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>NodeSelector identifies nodes to create VMs on</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchLabels</code><br>
<em>
map[string]string
</em>
</td>
<td>
<p>Node type needs to specified</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.NodeSet">NodeSet
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.VinoSpec">VinoSpec</a>)
</p>
<p>NodeSet node definitions</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Parameter for Node control-plane or worker</p>
</td>
</tr>
<tr>
<td>
<code>count</code><br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bmhLabels</code><br>
<em>
map[string]string
</em>
</td>
<td>
<p>BMHLabels labels will be copied directly to BMHs that will be created
These labels will override keys from k8s node, that are specified in vino.NodeLabelKeysToCopy</p>
</td>
</tr>
<tr>
<td>
<code>libvirtTemplate</code><br>
<em>
<a href="#airship.airshipit.org/v1.NamespacedName">
NamespacedName
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>networkInterfaces</code><br>
<em>
<a href="#airship.airshipit.org/v1.NetworkInterface">
[]NetworkInterface
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>diskDrives</code><br>
<em>
<a href="#airship.airshipit.org/v1.DiskDrivesTemplate">
[]DiskDrivesTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>networkDataTemplate</code><br>
<em>
<a href="#airship.airshipit.org/v1.NamespacedName">
NamespacedName
</a>
</em>
</td>
<td>
<p>NetworkDataTemplate must have a template key</p>
</td>
</tr>
<tr>
<td>
<code>rootDeviceName</code><br>
<em>
string
</em>
</td>
<td>
<p>RootDeviceName is the root device for underlying VM, /dev/vda for example
default is /dev/vda</p>
</td>
</tr>
<tr>
<td>
<code>bootInterfaceName</code><br>
<em>
string
</em>
</td>
<td>
<p>BootInterfaceName interface name to use to boot virtual machines</p>
</td>
</tr>
<tr>
<td>
<code>enableVNC</code><br>
<em>
bool
</em>
</td>
<td>
<p>EnableVNC create VNC for graphical interaction with the VM that will be created.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.Range">Range
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.BuilderNetwork">BuilderNetwork</a>, 
<a href="#airship.airshipit.org/v1.IPPoolSpec">IPPoolSpec</a>)
</p>
<p>Range has (inclusive) bounds within a subnet from which IPs can be allocated</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>start</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>stop</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.VMRoutes">VMRoutes
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Network">Network</a>)
</p>
<p>VMRoutes defined</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>network</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>netmask</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gateway</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.Vino">Vino
</h3>
<p>Vino is the Schema for the vinoes API</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br>
<em>
<a href="#airship.airshipit.org/v1.VinoSpec">
VinoSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>nodeSelector</code><br>
<em>
<a href="#airship.airshipit.org/v1.NodeSelector">
NodeSelector
</a>
</em>
</td>
<td>
<p>Define nodelabel parameters</p>
</td>
</tr>
<tr>
<td>
<code>configuration</code><br>
<em>
<a href="#airship.airshipit.org/v1.CPUConfiguration">
CPUConfiguration
</a>
</em>
</td>
<td>
<p>Define CPU configuration</p>
</td>
</tr>
<tr>
<td>
<code>networks</code><br>
<em>
<a href="#airship.airshipit.org/v1.Network">
[]Network
</a>
</em>
</td>
<td>
<p>Define network parameters</p>
</td>
</tr>
<tr>
<td>
<code>nodes</code><br>
<em>
<a href="#airship.airshipit.org/v1.NodeSet">
[]NodeSet
</a>
</em>
</td>
<td>
<p>Define node details</p>
</td>
</tr>
<tr>
<td>
<code>daemonSetOptions</code><br>
<em>
<a href="#airship.airshipit.org/v1.DaemonSetOptions">
DaemonSetOptions
</a>
</em>
</td>
<td>
<p>DaemonSetOptions defines how vino will spawn daemonset on nodes</p>
</td>
</tr>
<tr>
<td>
<code>bmcCredentials</code><br>
<em>
<a href="#airship.airshipit.org/v1.BMCCredentials">
BMCCredentials
</a>
</em>
</td>
<td>
<p>BMCCredentials contain credentials that will be used to create BMH nodes
sushy tools will use these credentials as well, to set up authentication</p>
</td>
</tr>
<tr>
<td>
<code>nodeLabelKeysToCopy</code><br>
<em>
[]string
</em>
</td>
<td>
<p>NodeLabelKeysToCopy vino controller will get these labels from k8s nodes
and place them on BMHs that correspond to this node</p>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHost</code><br>
<em>
string
</em>
</td>
<td>
<p>PXEBootImageHost will be used to download the PXE boot image</p>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHostPort</code><br>
<em>
int
</em>
</td>
<td>
<p>PXEBootImageHostPort will be used to download the PXE boot image</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br>
<em>
<a href="#airship.airshipit.org/v1.VinoStatus">
VinoStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.VinoSpec">VinoSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Vino">Vino</a>)
</p>
<p>VinoSpec defines the desired state of Vino</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeSelector</code><br>
<em>
<a href="#airship.airshipit.org/v1.NodeSelector">
NodeSelector
</a>
</em>
</td>
<td>
<p>Define nodelabel parameters</p>
</td>
</tr>
<tr>
<td>
<code>configuration</code><br>
<em>
<a href="#airship.airshipit.org/v1.CPUConfiguration">
CPUConfiguration
</a>
</em>
</td>
<td>
<p>Define CPU configuration</p>
</td>
</tr>
<tr>
<td>
<code>networks</code><br>
<em>
<a href="#airship.airshipit.org/v1.Network">
[]Network
</a>
</em>
</td>
<td>
<p>Define network parameters</p>
</td>
</tr>
<tr>
<td>
<code>nodes</code><br>
<em>
<a href="#airship.airshipit.org/v1.NodeSet">
[]NodeSet
</a>
</em>
</td>
<td>
<p>Define node details</p>
</td>
</tr>
<tr>
<td>
<code>daemonSetOptions</code><br>
<em>
<a href="#airship.airshipit.org/v1.DaemonSetOptions">
DaemonSetOptions
</a>
</em>
</td>
<td>
<p>DaemonSetOptions defines how vino will spawn daemonset on nodes</p>
</td>
</tr>
<tr>
<td>
<code>bmcCredentials</code><br>
<em>
<a href="#airship.airshipit.org/v1.BMCCredentials">
BMCCredentials
</a>
</em>
</td>
<td>
<p>BMCCredentials contain credentials that will be used to create BMH nodes
sushy tools will use these credentials as well, to set up authentication</p>
</td>
</tr>
<tr>
<td>
<code>nodeLabelKeysToCopy</code><br>
<em>
[]string
</em>
</td>
<td>
<p>NodeLabelKeysToCopy vino controller will get these labels from k8s nodes
and place them on BMHs that correspond to this node</p>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHost</code><br>
<em>
string
</em>
</td>
<td>
<p>PXEBootImageHost will be used to download the PXE boot image</p>
</td>
</tr>
<tr>
<td>
<code>pxeBootImageHostPort</code><br>
<em>
int
</em>
</td>
<td>
<p>PXEBootImageHostPort will be used to download the PXE boot image</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="airship.airshipit.org/v1.VinoStatus">VinoStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#airship.airshipit.org/v1.Vino">Vino</a>)
</p>
<p>VinoStatus defines the observed state of Vino</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>configMapRef</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<div class="admonition note">
<p class="last">This page was automatically generated with <code>gen-crd-api-reference-docs</code></p>
</div>
