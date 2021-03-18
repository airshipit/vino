/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VinoLabel is a vino base label
	VinoLabel = "vino.airshipit.org"
	// VinoLabelDSNameSelector used to label pods in daemon set to avoid collisions
	VinoLabelDSNameSelector = VinoLabel + "/" + "cr-name"
	// VinoLabelDSNamespaceSelector used to label pods in daemon set to avoid collisions
	VinoLabelDSNamespaceSelector = VinoLabel + "/" + "cr-namespace"
	// VinoFinalizer constant
	VinoFinalizer = "vino.airshipit.org"
	// EnvVarVMInterfaceName environment variable that is used to find VM interface to use for vms
	EnvVarVMInterfaceName = "VM_BRIDGE_INTERFACE"
)

// VinoSpec defines the desired state of Vino
type VinoSpec struct {
	// Define nodelabel parameters
	NodeSelector *NodeSelector `json:"nodeSelector,omitempty"`
	// Define CPU configuration
	CPUConfiguration *CPUConfiguration `json:"configuration,omitempty"`
	// Define network parameters
	Networks []Network `json:"networks,omitempty"`
	// Define node details
	Nodes []NodeSet `json:"nodes,omitempty"`
	// DaemonSetOptions defines how vino will spawn daemonset on nodes
	DaemonSetOptions DaemonSetOptions `json:"daemonSetOptions,omitempty"`
	// VMBridge defines the single interface name to be used as a bridge for VMs
	VMBridge string `json:"vmBridge"`
	// BMCCredentials contain credentials that will be used to create BMH nodes
	// sushy tools will use these credentials as well, to set up authentication
	BMCCredentials BMCCredentials `json:"bmcCredentials"`
}

// BMCCredentials contain credentials that will be used to create BMH nodes
// sushy tools will use these credentials as well, to set up authentication
type BMCCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NodeSelector identifies nodes to create VMs on
type NodeSelector struct {
	// Node type needs to specified
	MatchLabels map[string]string `json:"matchLabels"`
}

// CPUConfiguration CPU node configuration
type CPUConfiguration struct {
	//Exclude CPU example 0-4,54-60
	CPUExclude string `json:"cpuExclude,omitempty"`
}

// Network defines libvirt networks
type Network struct {
	//Network Parameter defined
	Name            string     `json:"name,omitempty"`
	SubNet          string     `json:"subnet,omitempty"`
	Type            string     `json:"type,omitempty"`
	AllocationStart string     `json:"allocationStart,omitempty"`
	AllocationStop  string     `json:"allocationStop,omitempty"`
	DNSServers      []string   `json:"dns_servers,omitempty"`
	Routes          []VMRoutes `json:"routes,omitempty"`
	// MACPrefix defines the zero-padded MAC prefix to use for
	// VM mac addresses, and is the first address that will be
	// allocated sequentially to VMs in this network.
	// If omitted, a default private MAC prefix will be used.
	// The prefix should be specified in full MAC notation, e.g.
	// 06:42:42:00:00:00
	MACPrefix string `json:"macPrefix,omitempty"`
}

// VMRoutes defined
type VMRoutes struct {
	Network string `json:"network,omitempty"`
	Netmask string `json:"netmask,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

//NodeSet node definitions
type NodeSet struct {
	//Parameter for Node master or worker-standard
	Name                      string              `json:"name,omitempty"`
	Count                     int                 `json:"count,omitempty"`
	NodeLabel                 VMNodeFlavor        `json:"labels,omitempty"`
	LibvirtTemplateDefinition NamespacedName      `json:"libvirtTemplate,omitempty"`
	NetworkInterfaces         []NetworkInterface  `json:"networkInterfaces,omitempty"`
	DiskDrives                *DiskDrivesTemplate `json:"diskDrives,omitempty"`
	// NetworkDataTemplate must have a template key
	NetworkDataTemplate NamespacedName `json:"networkDataTemplate,omitempty"`
}

// VMNodeFlavor labels for node to be annotated
type VMNodeFlavor struct {
	VMFlavor map[string]string `json:"vmFlavor,omitempty"`
}

// NamespacedName to be used to spawn VMs
type NamespacedName struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// DaemonSetOptions be used to spawn vino-builder, libvirt, sushy an
type DaemonSetOptions struct {
	Template     NamespacedName `json:"namespacedName,omitempty"`
	LibvirtImage string         `json:"libvirtImage,omitempty"`
	SushyImage   string         `json:"sushyImage,omitempty"`
	VinoBuilder  string         `json:"vinoBuilderImage,omitempty"`
	NodeLabeler  string         `json:"nodeAnnotatorImage,omitempty"`
}

// NetworkInterface define interface on the VM
type NetworkInterface struct {
	// Define parameter for network interfaces
	Name        string            `json:"name,omitempty"`
	Type        string            `json:"type,omitempty"`
	NetworkName string            `json:"network,omitempty"`
	MTU         int               `json:"mtu,omitempty"`
	Options     map[string]string `json:"options,omitempty"`
}

// DiskDrivesTemplate defines disks on the VM
type DiskDrivesTemplate struct {
	Name    string       `json:"name,omitempty"`
	Type    string       `json:"type,omitempty"`
	Path    string       `json:"path,omitempty"`
	Options *DiskOptions `json:"options,omitempty"`
}

// DiskOptions disk options
type DiskOptions struct {
	SizeGB int  `json:"sizeGb,omitempty"`
	Sparse bool `json:"sparse,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Vino is the Schema for the vinoes API
type Vino struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VinoSpec   `json:"spec,omitempty"`
	Status VinoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VinoList contains a list of Vino
type VinoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vino `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Vino{}, &VinoList{})
}

// VinoStatus defines the observed state of Vino
type VinoStatus struct {
	ConfigMapRef corev1.ObjectReference `json:"configMapRef,omitempty"`
	Conditions   []metav1.Condition     `json:"conditions,omitempty"`
}

// VinoProgressing registers progress toward reconciling the given Vino
// by resetting the status to a progressing state.
func VinoProgressing(v *Vino) {
	v.Status.Conditions = []metav1.Condition{}
	apimeta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionFalse,
		Reason:             ProgressingReason,
		Message:            "Reconciliation progressing",
		Type:               ConditionTypeReady,
		ObservedGeneration: v.GetGeneration(),
	})
}

// VinoReady registers success reconciling the given Vino.
func VinoReady(v *Vino) {
	apimeta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             ReconciliationSucceededReason,
		Message:            "Reconciliation succeeded",
		Type:               ConditionTypeReady,
		ObservedGeneration: v.GetGeneration(),
	})
}
