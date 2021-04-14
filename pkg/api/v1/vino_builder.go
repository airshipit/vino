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
	GWIPBridge string    `json:"gwIPBridge,omitempty"`
	Networks   []Network `json:"networks,omitempty"`
	Nodes      []NodeSet `json:"nodes,omitempty"`
	// (TODO) change json tag to cpuConfiguration when vino-builder has these chanages as well
	CPUConfiguration CPUConfiguration         `json:"configuration,omitempty"`
	Domains          map[string]BuilderDomain `json:"domains,omitempty"`
}

type BuilderNetworkInterface struct {
	MACAddress string `json:"macAddress,omitempty"`
}

// BuilderDomain represents a VINO libvirt domain
type BuilderDomain struct {
	Interfaces map[string]BuilderNetworkInterface `json:"interfaces,omitempty"`
}
