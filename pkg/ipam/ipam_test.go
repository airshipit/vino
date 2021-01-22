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
	"math"
	"testing"

	vinov1 "vino/pkg/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocateIP(t *testing.T) {
	tests := []struct {
		name, subnet, expectedErr string
		subnetRange               vinov1.Range
	}{
		{
			name:        "success ipv4",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.9"},
		},
		{
			name:        "success ipv6",
			subnet:      "2600:1700:b030:0000::/72",
			subnetRange: vinov1.Range{Start: "2600:1700:b030:0000::", Stop: "2600:1700:b030:0009::"},
		},
		{
			name:        "error subnet not allocated ipv4",
			subnet:      "10.0.0.0/20",
			subnetRange: vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.9"},
			expectedErr: "IPAM subnet 10.0.0.0/20 not allocated",
		},
		{
			name:        "error subnet not allocated ipv6",
			subnet:      "2600:1700:b030:0000::/80",
			subnetRange: vinov1.Range{Start: "2600:1700:b030:0000::", Stop: "2600:1700:b030:0009::"},
			expectedErr: "IPAM subnet 2600:1700:b030:0000::/80 not allocated",
		},
		{
			name:        "error range not allocated ipv4",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.2.0", Stop: "10.0.2.9"},
			expectedErr: "IPAM range [10.0.2.0,10.0.2.9] in subnet 10.0.0.0/16 is not allocated",
		},
		{
			name:        "error range not allocated ipv6",
			subnet:      "2600:1700:b030:0000::/72",
			subnetRange: vinov1.Range{Start: "2600:1700:b030:0000::", Stop: "2600:1700:b030:1111::"},
			expectedErr: "IPAM range [2600:1700:b030:0000::,2600:1700:b030:1111::] " +
				"in subnet 2600:1700:b030:0000::/72 is not allocated",
		},
		{
			name:        "error range exhausted ipv4",
			subnet:      "192.168.0.0/1",
			subnetRange: vinov1.Range{Start: "192.168.0.0", Stop: "192.168.0.0"},
			expectedErr: "IPAM range [192.168.0.0,192.168.0.0] in subnet 192.168.0.0/1 is exhausted",
		},
		{
			name:        "error range exhausted ipv6",
			subnet:      "2600:1700:b031:0000::/64",
			subnetRange: vinov1.Range{Start: "2600:1700:b031:0000::", Stop: "2600:1700:b031:0000::"},
			expectedErr: "IPAM range [2600:1700:b031:0000::,2600:1700:b031:0000::] " +
				"in subnet 2600:1700:b031:0000::/64 is exhausted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ipammer := NewIpam()

			// Pre-populate IPAM with some precondition test data
			err := ipammer.AddSubnetRange("10.0.0.0/16", vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.9"})
			require.NoError(t, err)
			err = ipammer.AddSubnetRange("2600:1700:b030:0000::/72",
				vinov1.Range{Start: "2600:1700:b030:0000::", Stop: "2600:1700:b030:0009::"})
			require.NoError(t, err)
			err = ipammer.AddSubnetRange("192.168.0.0/1",
				vinov1.Range{Start: "192.168.0.0", Stop: "192.168.0.0"})
			require.NoError(t, err)
			err = ipammer.AddSubnetRange("2600:1700:b031:0000::/64",
				vinov1.Range{Start: "2600:1700:b031:0000::", Stop: "2600:1700:b031:0000::"})
			require.NoError(t, err)
			_, err = ipammer.AllocateIP("192.168.0.0/1", vinov1.Range{Start: "192.168.0.0", Stop: "192.168.0.0"})
			require.NoError(t, err)
			_, err = ipammer.AllocateIP("2600:1700:b031:0000::/64",
				vinov1.Range{Start: "2600:1700:b031:0000::", Stop: "2600:1700:b031:0000::"})
			require.NoError(t, err)
			ip, err := ipammer.AllocateIP(tt.subnet, tt.subnetRange)
			if tt.expectedErr != "" {
				assert.Equal(t, "", ip)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, ip)
			}
		})
	}
}

func TestNewRange(t *testing.T) {
	tests := []struct {
		name, start, stop, expectedErr string
	}{
		{
			name:        "success",
			start:       "10.0.0.1",
			stop:        "10.0.0.2",
			expectedErr: "",
		},
		{
			name:        "error stop less than start",
			start:       "10.0.0.2",
			stop:        "10.0.0.1",
			expectedErr: "is invalid",
		},
		{
			name:        "error bad start",
			start:       "10.0.0.2.x",
			stop:        "10.0.0.1",
			expectedErr: "is invalid",
		},
		{
			name:        "error bad stop",
			start:       "10.0.0.2",
			stop:        "10.0.0.1.x",
			expectedErr: "is invalid",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRange(tt.start, tt.stop)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, r.Start, tt.start)
				assert.Equal(t, r.Stop, tt.stop)
			}
		})
	}
}

// Test some error handling that is not captured by TestAllocateIP
func TestAddSubnetRange(t *testing.T) {
	tests := []struct {
		name, subnet, expectedErr string
		subnetRange               vinov1.Range
	}{
		{
			name:        "success",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.2.0", Stop: "10.0.2.9"},
			expectedErr: "",
		},
		{
			name:        "error range already exists",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.9"},
			expectedErr: "IPAM range [10.0.1.0,10.0.1.9] in subnet 10.0.0.0/16 overlaps",
		},
		// TODO: check for partially overlapping ranges and subnets
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ipammer := NewIpam()

			// Pre-populate IPAM with some precondition test data
			err := ipammer.AddSubnetRange("10.0.0.0/16", vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.9"})
			require.NoError(t, err)
			err = ipammer.AddSubnetRange(tt.subnet, tt.subnetRange)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
func TestFindFreeIPInRange(t *testing.T) {
	tests := []struct {
		name        string
		subnet      string
		subnetRange vinov1.Range
		out         string
		expectedErr string
	}{
		{
			name:        "ip available IPv4",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.1.0", Stop: "10.0.1.10"},
			out:         "10.0.1.0",
		},
		{
			name:        "ip unavailable IPv4",
			subnet:      "10.0.0.0/16",
			subnetRange: vinov1.Range{Start: "10.0.2.0", Stop: "10.0.2.0"},
			out:         "",
			expectedErr: "IPAM range [10.0.2.0,10.0.2.0] in subnet 10.0.0.0/16 is exhausted",
		},
		{
			name:        "ip available IPv6",
			subnet:      "2600:1700:b030:0000::/64",
			subnetRange: vinov1.Range{Start: "2600:1700:b030:1001::", Stop: "2600:1700:b030:1009::"},
			out:         "2600:1700:b030:1001::",
		},
		{
			name:        "ip unavailable IPv6",
			subnet:      "2600:1700:b031::/64",
			subnetRange: vinov1.Range{Start: "2600:1700:b031::", Stop: "2600:1700:b031::"},
			expectedErr: "IPAM range [2600:1700:b031::,2600:1700:b031::] " +
				"in subnet 2600:1700:b031::/64 is exhausted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ippool := vinov1.IPPoolSpec{
				Subnet: tt.subnet,
				// One available and one unavailable range each for ipv4/6
				Ranges: []vinov1.Range{
					{Start: "10.0.1.0", Stop: "10.0.1.10"},
					{Start: "10.0.2.0", Stop: "10.0.2.0"},
					{Start: "2600:1700:b030:1001::", Stop: "2600:1700:b030:1009::"},
					{Start: "2600:1700:b031::", Stop: "2600:1700:b031::"},
				},
				AllocatedIPs: []string{"10.0.2.0", "2600:1700:b031::"},
			}
			actual, err := findFreeIPInRange(&ippool, tt.subnetRange)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.out, actual)
			}
		})
	}
}

func TestSliceToMap(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  map[string]struct{}
	}{
		{
			name: "empty slice",
			in:   []string{},
			out:  map[string]struct{}{},
		},
		{
			name: "one-element slice",
			in:   []string{"foo"},
			out:  map[string]struct{}{"foo": {}},
		},
		{
			name: "two-element slice",
			in:   []string{"foo", "bar"},
			out:  map[string]struct{}{"foo": {}, "bar": {}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := sliceToMap(tt.in)
			assert.Equal(t, tt.out, actual)
		})
	}
}

func TestIPStringToInt(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		out         uint64
		expectedErr string
	}{
		{
			name: "valid IPv4 address",
			in:   "1.0.0.1",
			out:  uint64(math.Pow(2, 24) + 1),
		},
		{
			name:        "invalid IPv4 address",
			in:          "1.0.0.1.1",
			out:         0,
			expectedErr: " is invalid",
		},
		{
			name: "valid IPv6 address",
			in:   "0001:0000:0000:0001::",
			out:  uint64(math.Pow(2, 48) + 1),
		},
		{
			name:        "invalid IPv6 address",
			in:          "1000:0000:0000:foobar::",
			out:         0,
			expectedErr: " is invalid",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ipStringToInt(tt.in)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Empty(t, tt.out)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.out, actual)
			}
		})
	}
}

func TestIntToByteArray(t *testing.T) {
	tests := []struct {
		name string
		in   uint64
		out  []byte
	}{
		{
			name: "zeros",
			in:   0,
			out:  make([]byte, 8),
		},
		{
			name: "IPv4 255's",
			in:   uint64(math.Pow(2, 32) - 1),
			out:  []byte{0, 0, 0, 0, 255, 255, 255, 255},
		},
		{
			name: "value in the middle",
			in:   512,
			out:  []byte{0, 0, 0, 0, 0, 0, 2, 0},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := intToByteArray(tt.in)
			assert.Equal(t, tt.out, actual)
		})
	}
}

func TestByteArrayToInt(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		out  uint64
	}{
		{
			name: "zeros",
			in:   make([]byte, 8),
			out:  0,
		},
		{
			name: "255's",
			in:   []byte{0, 0, 0, 0, 255, 255, 255, 255},
			out:  uint64(math.Pow(2, 32) - 1),
		},
		{
			name: "value in the middle",
			in:   []byte{0, 0, 0, 0, 0, 0, 2, 0},
			out:  512,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := byteArrayToInt(tt.in)
			assert.Equal(t, tt.out, actual)
		})
	}
}
