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

// TODO (kkalynovskyi) create an API object for this, and refactor vino-builder to read it from kubernetes.
type Builder struct {
	GWIPBridge           string `json:"gwIPBridge,omitempty"`
	PXEBootImageHost     string `json:"pxeBootImageHost,omitempty"`
	PXEBootImageHostPort int    `json:"pxeBootImageHostPort,omitempty"`

	Networks []Network `json:"networks,omitempty"`
	// (TODO) change json tag to cpuConfiguration when vino-builder has these chanages as well
	CPUConfiguration CPUConfiguration `json:"configuration,omitempty"`
	Domains          []BuilderDomain  `json:"domains,omitempty"`
	NodeCount        int              `json:"nodeCount,omitempty"`
}

type BuilderNetworkInterface struct {
	IPAddress        string `json:"ipAddress,omitempty"`
	MACAddress       string `json:"macAddress,omitempty"`
	NetworkInterface `json:",inline"`
}

// BuilderDomain represents a VINO libvirt domain
type BuilderDomain struct {
	Name           string `json:"name,omitempty"`
	Role           string `json:"role,omitempty"`
	BootMACAddress string `json:"bootMACAddress,omitempty"`

	Interfaces []BuilderNetworkInterface `json:"interfaces,omitempty"`
}
