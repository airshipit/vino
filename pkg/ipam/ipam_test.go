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
	"math"
	"testing"
	vinov1 "vino/pkg/api/v1"
	test "vino/pkg/test"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Sets up a mock client that will serve up
func SetUpMockClient(ctx context.Context, ctrl *gomock.Controller) *test.MockClient {
	m := test.NewMockClient(ctrl)
	// Pre-populate IPAM with some precondition test data
	preExistingIpam := vinov1.IPPoolList{
		Items: []vinov1.IPPool{
			{
				Spec: vinov1.IPPoolSpec{
					Subnet: "10.0.0.0/16",
					Ranges: []vinov1.Range{
						{Start: "10.0.1.0", Stop: "10.0.1.9"},
					},
				},
			},
			{
				Spec: vinov1.IPPoolSpec{
					Subnet: "2600:1700:b030:0000::/72",
					Ranges: []vinov1.Range{
						{Start: "2600:1700:b030:0000::", Stop: "2600:1700:b030:0009::"},
					},
				},
			},
			{
				Spec: vinov1.IPPoolSpec{
					Subnet: "192.168.0.0/1",
					Ranges: []vinov1.Range{
						{Start: "192.168.0.0", Stop: "192.168.0.0"},
					},
					AllocatedIPs: []string{"192.168.0.0"},
				},
			},
			{
				Spec: vinov1.IPPoolSpec{
					Subnet: "2600:1700:b031:0000::/64",
					Ranges: []vinov1.Range{
						{Start: "2600:1700:b031:0000::", Stop: "2600:1700:b031:0000::"},
					},
					AllocatedIPs: []string{"2600:1700:b031:0000::"},
				},
			},
		},
	}

	m.EXPECT().List(ctx, gomock.Any(), gomock.Any()).SetArg(1, preExistingIpam)
	m.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Update(ctx, gomock.Any(), gomock.Any()).AnyTimes()
	return m
}

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := SetUpMockClient(ctx, ctrl)
			ipammer := NewIpam(log.Log, m, "vino-system")
			ipammer.Log = log.Log

			ip, err := ipammer.AllocateIP(ctx, tt.subnet, tt.subnetRange)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, "", ip)
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := SetUpMockClient(ctx, ctrl)
			ipammer := NewIpam(log.Log, m, "vino-system")

			err := ipammer.AddSubnetRange(ctx, tt.subnet, tt.subnetRange)
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
		out  map[uint64]struct{}
	}{
		{
			name: "empty slice",
			in:   []string{},
			out:  map[uint64]struct{}{},
		},
		{
			name: "one-element slice",
			in:   []string{"0.0.0.1"},
			out:  map[uint64]struct{}{1: {}},
		},
		{
			name: "two-element slice",
			in:   []string{"0.0.0.1", "0.0.0.2"},
			out:  map[uint64]struct{}{1: {}, 2: {}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual, err := sliceToMap(tt.in)
			assert.Equal(t, tt.out, actual)
			require.NoError(t, err)
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

func TestSubnetResourceName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "ipv4",
			in:   "192.168.0.0/24",
			out:  "ippool-192-168-0-0-24",
		},
		{
			name: "ipv6",
			in:   "0001:0000:0000:0001::/32",
			out:  "ippool-0001-0000-0000-0001---32",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := subnetResourceName(tt.in)
			assert.Equal(t, tt.out, actual)
		})
	}
}

func TestApplyIPPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := test.NewMockClient(ctrl)
	ipammer := NewIpam(log.Log, m, "vino-system")
	ctx := context.Background()

	spec := vinov1.IPPoolSpec{
		Subnet: "192.168.0.0/24",
		Ranges: []vinov1.Range{
			{
				Start: "192.168.1.10",
				Stop:  "192.168.1.20",
			},
		},
	}
	pool := vinov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "vino-system",
			Name:      "ippool-192-168-0-0-24",
		},
		Spec: spec,
	}
	emptyPool := &vinov1.IPPool{}

	// Test Create scenario
	m = test.NewMockClient(ctrl)
	ipammer.Client = m
	m.EXPECT().Get(ctx, client.ObjectKeyFromObject(&pool), emptyPool).Return(
		apierrors.NewNotFound(schema.GroupResource{
			Group: "airship.airshipit.org", Resource: "ippools"}, "ippool-192-168-0-0-24"))
	m.EXPECT().Create(ctx, &pool)
	err := ipammer.applyIPPool(ctx, spec)
	assert.NoError(t, err)

	// Test Update scenario
	existingPool := pool.DeepCopy()
	existingPool.Generation = 1
	m = test.NewMockClient(ctrl)
	ipammer.Client = m
	m.EXPECT().Get(ctx, client.ObjectKeyFromObject(&pool), emptyPool).SetArg(2, *existingPool)
	m.EXPECT().Update(ctx, &pool)
	err = ipammer.applyIPPool(ctx, spec)
	assert.NoError(t, err)

	// Test non-already-exists error scenario
	m = test.NewMockClient(ctrl)
	ipammer.Client = m
	m.EXPECT().Get(ctx, client.ObjectKeyFromObject(&pool), emptyPool).Return(
		apierrors.NewBadRequest("bad things happened"))
	err = ipammer.applyIPPool(ctx, spec)
	assert.Error(t, err)
}

func TestGetIPPools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	spec := vinov1.IPPoolSpec{
		Subnet: "192.168.0.0/24",
		Ranges: []vinov1.Range{
			{
				Start: "192.168.1.10",
				Stop:  "192.168.1.20",
			},
		},
	}
	pool := vinov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "vino-system",
			Name:      "ippool-192-168-0-0-24",
		},
		Spec: spec,
	}
	fullList := vinov1.IPPoolList{Items: []vinov1.IPPool{pool}}
	expectedResult := map[string]*vinov1.IPPoolSpec{"192.168.0.0/24": &spec}

	m := test.NewMockClient(ctrl)
	ipammer := NewIpam(log.Log, m, "vino-system")
	ipammer.Client = m
	m.EXPECT().List(ctx, gomock.Any(), client.InNamespace("vino-system")).SetArg(1, fullList)
	actualResult, err := ipammer.getIPPools(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, actualResult)
}
