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
	"context"
	"net"
	"regexp"
	"strings"
	"unsafe"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vinov1 "vino/pkg/api/v1"
)

// Ipam provides IPAM reservation, backed by IPPool CRs
type Ipam struct {
	Log       logr.Logger
	Client    client.Client
	Namespace string
}

// NewIpam initializes an empty IPAM configuration.
// TODO: add ability to remove IP addresses and ranges
func NewIpam(logger logr.Logger, client client.Client, namespace string) *Ipam {
	return &Ipam{
		Log:       logger,
		Client:    client,
		Namespace: namespace,
	}
}

// Create a new Range, validating its input
func NewRange(start string, stop string) (vinov1.Range, error) {
	r := vinov1.Range{Start: start, Stop: stop}
	a, e := ipStringToInt(start)
	if e != nil {
		return vinov1.Range{}, e
	}
	b, e := ipStringToInt(stop)
	if e != nil {
		return vinov1.Range{}, e
	}
	if b < a {
		return vinov1.Range{}, ErrSubnetRangeInvalid{r}
	}
	return r, nil
}

// AddSubnetRange adds a range within a subnet for IP allocation
// TODO error: range overlaps with existing range or subnet overlaps with existing subnet
// TODO error: invalid range for subnet
func (i *Ipam) AddSubnetRange(ctx context.Context, subnet string, subnetRange vinov1.Range) error {
	logger := i.Log.WithValues("subnet", subnet, "subnetRange", subnetRange)
	// Does the subnet already exist? (this is fine)
	ippools, err := i.getIPPools(ctx)
	if err != nil {
		return err
	}
	ippool, exists := ippools[subnet]
	if !exists {
		logger.Info("IPAM creating subnet and adding range")
		ippool = &vinov1.IPPoolSpec{
			Subnet:       subnet,
			Ranges:       []vinov1.Range{},
			AllocatedIPs: []string{},
		}
		ippools[subnet] = ippool
	} else {
		logger.Info("IPAM subnet already exists; adding range")
		// Does the subnet's requested range already exist? (this should fail)
		for _, r := range ippool.Ranges {
			if r == subnetRange {
				return ErrSubnetRangeOverlapsWithExistingRange{Subnet: subnet, SubnetRange: subnetRange}
			}
		}
	}
	ippool.Ranges = append(ippool.Ranges, subnetRange)
	err = i.applyIPPool(ctx, *ippool)
	if err != nil {
		return err
	}

	return nil
}

// AllocateIP allocates an IP from a range and return it
func (i *Ipam) AllocateIP(ctx context.Context, subnet string, subnetRange vinov1.Range) (string, error) {
	ippools, err := i.getIPPools(ctx)
	if err != nil {
		return "", err
	}
	ippool, exists := ippools[subnet]
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
	i.Log.Info("Allocating IP", "ip", ip, "subnet", subnet, "subnetRange", subnetRange)
	ippool.AllocatedIPs = append(ippool.AllocatedIPs, ip)
	err = i.applyIPPool(ctx, *ippool)
	if err != nil {
		return "", err
	}

	return ip, nil
}

// This converts IP ranges/addresses into iterable ints,
// steps through them looking for one that that is not already
// in use, converts it back to a string and returns it.
// It does not itself add it to the list of assigned IPs.
func findFreeIPInRange(ippool *vinov1.IPPoolSpec, subnetRange vinov1.Range) (string, error) {
	allocatedIPSet, err := sliceToMap(ippool.AllocatedIPs)
	if err != nil {
		return "", err
	}
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
		_, in := allocatedIPSet[ip]
		if !in {
			// Found an unallocated IP
			return intToString(ip), nil
		}
	}
	return "", ErrSubnetRangeExhausted{ippool.Subnet, subnetRange}
}

// Create a map[uint64]struct{} representation of a string slice,
// for efficient set lookups
func sliceToMap(slice []string) (map[uint64]struct{}, error) {
	m := map[uint64]struct{}{}
	for _, s := range slice {
		i, err := ipStringToInt(s)
		if err != nil {
			return m, err
		}
		m[i] = struct{}{}
	}
	return m, nil
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

// Transforms a subnet into k8s-friendly resource name
func subnetResourceName(subnet string) string {
	regex := regexp.MustCompile(`[:./]`)
	return "ippool-" + regex.ReplaceAllString(subnet, "-")
}

// Persist a pool to the API server (Create or Update)
func (i *Ipam) applyIPPool(ctx context.Context, spec vinov1.IPPoolSpec) error {
	logger := i.Log.WithValues("subnet", spec.Subnet)

	ippool := &vinov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      subnetResourceName(spec.Subnet),
		},
		Spec: spec,
	}
	existingPool := &vinov1.IPPool{}
	err := i.Client.Get(ctx, client.ObjectKeyFromObject(ippool), existingPool)
	if err != nil {
		// Is it an unexpected error?
		if !apierrors.IsNotFound(err) {
			return err
		}
		// The error is a warning that the resource doesn't exist yet, so we should create it
		logger.Info("IPAM creating IPPool")
		err = i.Client.Create(ctx, ippool)
	} else {
		logger.Info("IPAM IPPool already exists; updating it")
		ippool.ObjectMeta.ResourceVersion = existingPool.ObjectMeta.ResourceVersion
		err = i.Client.Update(ctx, ippool)
	}
	if err != nil {
		return err
	}
	return err
}

// Return a mapping of all allocated subnets to their IPPoolSpecs.
func (i *Ipam) getIPPools(ctx context.Context) (map[string]*vinov1.IPPoolSpec, error) {
	list := &vinov1.IPPoolList{}
	err := i.Client.List(ctx, list, client.InNamespace(i.Namespace))
	ippools := make(map[string]*vinov1.IPPoolSpec)
	if err != nil {
		return map[string]*vinov1.IPPoolSpec{}, err
	}
	for _, ippool := range list.Items {
		ippools[ippool.Spec.Subnet] = ippool.Spec.DeepCopy()
	}
	return ippools, nil
}
