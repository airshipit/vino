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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IPPoolSpec tracks allocation ranges and statuses within a specific
// subnet IPv4 or IPv6 subnet.  It has a set of ranges of IPs
// within the subnet from which IPs can be allocated by IPAM,
// and a set of IPs that are currently allocated already.
type IPPoolSpec struct {
	Subnet          string           `json:"subnet"`
	Ranges          []Range          `json:"ranges"`
	AllocatedRanges []AllocatedRange `json:"allocatedRanges,omitempty"`
	AllocatedIPs    []AllocatedIP    `json:"allocatedIPs"`
	// MACPrefix defines the MAC prefix to use for VM mac addresses
	MACPrefix string `json:"macPrefix"`
	// NextMAC indicates the next MAC address (in sequence) that
	// will be provisioned to a VM in this Subnet
	NextMAC string `json:"nextMAC"`
}

// AllocatedIP Allocates an IP and MAC address to an entity
type AllocatedIP struct {
	IP          string `json:"ip"`
	MAC         string `json:"mac"`
	AllocatedTo string `json:"allocatedTo"`
}

type AllocatedRange struct {
	AllocatedTo string `json:"allocatedTo"`
	Range       `json:",inline"`
}

// Range has (inclusive) bounds within a subnet from which IPs can be allocated
type Range struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

// IPPoolStatus defines the observed state of IPPool
type IPPoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// IPPool is the Schema for the ippools API
type IPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPPoolSpec   `json:"spec,omitempty"`
	Status IPPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPPoolList contains a list of IPPool
type IPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPPool{}, &IPPoolList{})
}
