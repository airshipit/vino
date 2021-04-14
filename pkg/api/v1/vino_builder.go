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

type Builder struct {
	GWIPBridge string                    `json:"gwIPBridge,omitempty"`
	Networks   map[string]BuilderNetwork `json:"networks,omitempty"`
	Domains    map[string]BuilderDomain  `json:"domains,omitempty"`
}

type BuilderNetworkInterface struct {
	MACAddress string `json:"macAddress,omitempty"`
}

type BuilderNetwork struct {
	// Placeholder for future development
}

type BuilderDomain struct {
	Interfaces map[string]BuilderNetworkInterface `json:"interfaces,omitempty"`
}
