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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VinoSpec defines the desired state of Vino
type VinoSpec struct {
	//Define nodelabel parameters
	NodeSelector *NodeSelector `json:"nodelabels,omitempty"`
	//Define CPU configuration
	Configuration *CPUConfiguration `json:"configuration,omitempty"`
	//Define network Parametes
	Network *Network `json:"networks,omitempty"`
	//Define node details
	Node []NodeSet `json:"nodes,omitempty"`
}

type NodeSelector struct {
	// Node type needs to specified
	MatchLabels map[string]string `json:"matchLabels"`
}
type CPUConfiguration struct {
	//Exclude CPU example 0-4,54-60
	CPUExclude string `json:"cpuExclude,omitempty"`
}

//Define network specs
type Network struct {
	//Network Parameter defined
	Name            string    `json:"name,omitempty"`
	SubNet          string    `json:"subnet,omitempty"`
	AllocationStart string    `json:"allocationStart,omitempty"`
	AllocationStop  string    `json:"allocationStop,omitempty"`
	DNSServers      []string  `json:"dns_servers,omitempty"`
	Routes          *VMRoutes `json:"routes,omitempty"`
}

//Routes defined
type VMRoutes struct {
	To  string `json:"to,omitempty"`
	Via string `json:"via,omitempty"`
}

//VinoSpec node definitions
type NodeSet struct {

	//Parameter for Node master or worker-standard
	Name                      string              `json:"name,omitempty"`
	NodeLabel                 *VMNodeFlavor       `json:"labels,omitempty"`
	Count                     int                 `json:"count,omitempty"`
	LibvirtTemplateDefinition *LibvirtTemplate    `json:"libvirtTemplateDefinition,omitempty"`
	NetworkInterface          *NetworkInterface   `json:"networkInterfaces,omitempty"`
	DiskDrives                *DiskDrivesTemplate `json:"diskDrives,omitempty"`
}

//Define node flavor
type VMNodeFlavor struct {
	VMFlavor map[string]string `json:"vmFlavor,omitempty"`
}

//Define Libvirt template
type LibvirtTemplate struct {
	Name      string `json:"Name,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type NetworkInterface struct {

	//Define parameter for netwok interfaces
	Name        string            `json:"name,omitempty"`
	Type        string            `json:"type,omitempty"`
	NetworkName string            `json:"network,omitempty"`
	MTU         int               `json:"mtu,omitempty"`
	Options     *InterfaceOptions `json:"options,omitempty"`
}

//VinoSpec Network option parameter definition
type InterfaceOptions struct {
	InterfaceName []string          `json:"interfaceName,omitempty"`
	BridgeName    map[string]string `json:"bridgeName,omitempty"`
	Vlan          int               `json:"vlan,omitempty"`
}

//Define disk drive for the nodes
type DiskDrivesTemplate struct {
	Name    string       `json:"name,omitempty"`
	Type    string       `json:"type,omitempty"`
	Path    string       `json:"path,omitempty"`
	Options *DiskOptions `json:"options,omitempty"`
}

//Define disk size
type DiskOptions struct {
	SizeGB int  `json:"sizeGb,omitempty"`
	Sparse bool `json:"sparse,omitempty"`
}

// VinoStatus defines the observed state of Vino
type VinoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
}

// +kubebuilder:object:root=true

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
