/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package ipam

import (
	"fmt"
)

// ErrSubnetNotAllocated returned if the subnet is not registered in IPAM
type ErrSubnetNotAllocated struct {
	Subnet string
}

// ErrSubnetRangeOverlapsWithExistingRange returned if the subnet's range
// overlaps (partially or completely) with an already added range in that subnet
type ErrSubnetRangeOverlapsWithExistingRange struct {
	Subnet      string
	SubnetRange Range
}

// ErrSubnetRangeNotAllocated returned if the subnet's range is not registered in IPAM
type ErrSubnetRangeNotAllocated struct {
	Subnet      string
	SubnetRange Range
}

// ErrSubnetRangeExhausted returned if the subnet's range has no unallocated IPs
type ErrSubnetRangeExhausted struct {
	Subnet      string
	SubnetRange Range
}

// ErrInvalidIPAddress returned if an IP address string is malformed
type ErrInvalidIPAddress struct {
	IP string
}

// ErrNotSupported returned if unsupported address types are used
type ErrNotSupported struct {
	Message string
}

func (e ErrSubnetNotAllocated) Error() string {
	return fmt.Sprintf("IPAM subnet %s not allocated", e.Subnet)
}

func (e ErrSubnetRangeOverlapsWithExistingRange) Error() string {
	return fmt.Sprintf("IPAM range [%s,%s] in subnet %s overlaps with an existing range",
		e.SubnetRange.Start, e.SubnetRange.Stop, e.Subnet)
}

func (e ErrSubnetRangeNotAllocated) Error() string {
	return fmt.Sprintf("IPAM range [%s,%s] in subnet %s is not allocated",
		e.SubnetRange.Start, e.SubnetRange.Stop, e.Subnet)
}

func (e ErrSubnetRangeExhausted) Error() string {
	return fmt.Sprintf("IPAM range [%s,%s] in subnet %s is exhausted",
		e.SubnetRange.Start, e.SubnetRange.Stop, e.Subnet)
}

func (e ErrInvalidIPAddress) Error() string {
	return fmt.Sprintf("IP address %s is invalid", e.IP)
}

func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("%s", e.Message)
}
