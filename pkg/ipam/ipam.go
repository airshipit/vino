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
	"net"
	"strings"
	"unsafe"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ipam provides IPAM reservation, backed by IPPool CRs
type Ipam struct {
	Log    logr.Logger
	Scheme *runtime.Scheme
	Client client.Client

	ippools map[string]*IPPool
}

// IPPool tracks allocation ranges and statuses within a specific
// subnet IPv4 or IPv6 subnet.  It has a set of ranges of IPs
// within the subnet from which IPs can be allocated by IPAM,
// and a set of IPs that are currently allocated already.
type IPPool struct {
	Subnet       string
	Ranges       []Range
	AllocatedIPs []string
}

// Range has (inclusive) bounds within a subnet from which IPs can be allocated
type Range struct {
	Start string
	Stop  string
}

// NewIpam initializes an empty IPAM configuration.
// TODO: persist and refresh state from the API server
// TODO: add ability to remove IP addresses and ranges
func NewIpam() *Ipam {
	ippools := make(map[string]*IPPool)
	return &Ipam{
		ippools: ippools,
	}
}

// AddSubnetRange adds a range within a subnet for IP allocation
// TODO error: invalid range for subnet
// TODO error: range overlaps with existing range or subnet overlaps with existing subnet
func (i *Ipam) AddSubnetRange(subnet string, subnetRange Range) error {
	// Does the subnet already exist? (this is fine)
	ippool, exists := i.ippools[subnet]
	if !exists {
		ippool = &IPPool{
			Subnet:       subnet,
			Ranges:       []Range{subnetRange}, // TODO DeepCopy()
			AllocatedIPs: []string{},
		}
		i.ippools[subnet] = ippool
	} else {
		// Does the subnet's requested range already exist? (this should fail)
		exists = false
		for _, r := range ippool.Ranges {
			if r == subnetRange {
				exists = true
				break
			}
		}
		if exists {
			return ErrSubnetRangeOverlapsWithExistingRange{Subnet: subnet, SubnetRange: subnetRange}
		}
	}
	ippool.Ranges = append(ippool.Ranges, subnetRange)

	return nil
}

// AllocateIP allocates an IP from a range and return it
func (i *Ipam) AllocateIP(subnet string, subnetRange Range) (string, error) {
	// NOTE/TODO: this is not threadsafe, which is fine because
	// the final impl will use the api server as the backend, not local.
	ippool, exists := i.ippools[subnet]
	if !exists {
		return "", ErrSubnetNotAllocated{Subnet: subnet}
	}
	// Make sure the range has been allocated within the subnet
	var match bool
	for _, r := range ippool.Ranges {
		if r == subnetRange {
			match = true
			break
		}
	}
	if !match {
		return "", ErrSubnetRangeNotAllocated{Subnet: subnet, SubnetRange: subnetRange}
	}

	ip, err := findFreeIPInRange(ippool, subnetRange)
	if err != nil {
		return "", err
	}
	ippool.AllocatedIPs = append(ippool.AllocatedIPs, ip)
	return ip, nil
}

// This converts IP ranges/addresses into iterable ints,
// steps through them looking for one that that is not already
// in use, converts it back to a string and returns it.
// It does not itself add it to the list of assigned IPs.
func findFreeIPInRange(ippool *IPPool, subnetRange Range) (string, error) {
	allocatedIPSet := sliceToMap(ippool.AllocatedIPs)
	intToString := intToIPv4String
	if strings.Contains(ippool.Subnet, ":") {
		intToString = intToIPv6String
	}

	// Step through the range looking for free IPs
	start, err := ipStringToInt(subnetRange.Start)
	if err != nil {
		return "", err
	}
	stop, err := ipStringToInt(subnetRange.Stop)
	if err != nil {
		return "", err
	}

	for ip := start; ip <= stop; ip++ {
		_, in := allocatedIPSet[intToString(ip)]
		if !in {
			// Found an unallocated IP
			return intToString(ip), nil
		}
	}
	return "", ErrSubnetRangeExhausted{ippool.Subnet, subnetRange}
}

// Create a map[string]struct{} representation of a string slice,
// for efficient set lookups
func sliceToMap(slice []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return m
}

// Convert an IPV4 or IPV6 address string to an easily iterable uint64.
// For IPV4 addresses, this captures the full address (padding the MSB with 0's)
// For IPV6 addresses, this captures the most significant 8 bytes,
// and excludes the 8-byte interface identifier.
func ipStringToInt(ipString string) (uint64, error) {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return 0, ErrInvalidIPAddress{ipString}
	}

	var bytes []byte
	if ip.To4() != nil {
		// IPv4
		bytes = append(make([]byte, 4), ip.To4()...)
	} else {
		// IPv6
		bytes = ip.To16()[:8]
	}

	return byteArrayToInt(bytes), nil
}

func intToIPv4String(i uint64) string {
	bytes := intToByteArray(i)
	ip := net.IPv4(bytes[4], bytes[5], bytes[6], bytes[7])
	return ip.String()
}

func intToIPv6String(i uint64) string {
	// Pad with 8 more bytes of zeros on the right for the hosts's interface,
	// which will not be determined by IPAM.
	bytes := append(intToByteArray(i), make([]byte, 8)...)
	var ip net.IP = bytes
	return ip.String()
}

// Convert an uint64 into 8 bytes, with most significant byte first
// Based on https://gist.github.com/ecoshub/5be18dc63ac64f3792693bb94f00662f
func intToByteArray(num uint64) []byte {
	size := 8
	arr := make([]byte, size)
	for i := 0; i < size; i++ {
		byt := *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(i)))
		arr[size-i-1] = byt
	}
	return arr
}

// Convert an 8-byte array to an uint64
// Based on https://gist.github.com/ecoshub/5be18dc63ac64f3792693bb94f00662f
func byteArrayToInt(arr []byte) uint64 {
	val := uint64(0)
	size := 8
	for i := 0; i < size; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(size-i-1))) = arr[i]
	}
	return val
}
