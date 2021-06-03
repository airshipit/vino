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

package managers

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/go-logr/logr"
	metal3 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	vinov1 "vino/pkg/api/v1"
	"vino/pkg/ipam"
)

const (
	// DefaultMACPrefix is a private RFC 1918 MAC range used if
	// no MACPrefix is specified for a network in the ViNO CR
	DefaultMACPrefix = "02:00:00:00:00:00"
)

type networkTemplateValues struct {
	BMHName string

	Node     vinov1.NodeSet // the specific node type to be templated
	Networks []vinov1.Network
	vinov1.BuilderDomain
}

type BMHManager struct {
	Namespace string

	client.Client
	ViNO        *vinov1.Vino
	BootNetwork *vinov1.Network
	Ipam        *ipam.Ipam
	Logger      logr.Logger

	bmhList           []*metal3.BareMetalHost
	networkSecrets    []*corev1.Secret
	credentialSecrets []*corev1.Secret
}

func (r *BMHManager) ScheduleVMs(ctx context.Context) error {
	return r.requestVMs(ctx)
}

func (r *BMHManager) CreateBMHs(ctx context.Context) error {
	for _, secret := range r.networkSecrets {
		objKey := client.ObjectKeyFromObject(secret)
		r.Logger.Info("Applying network secret", "secret", objKey)
		if err := applyRuntimeObject(ctx, objKey, secret, r.Client); err != nil {
			return err
		}
	}

	for _, secret := range r.credentialSecrets {
		objKey := client.ObjectKeyFromObject(secret)
		r.Logger.Info("Applying network secret", "secret", objKey)
		if err := applyRuntimeObject(ctx, objKey, secret, r.Client); err != nil {
			return err
		}
	}

	for _, bmh := range r.bmhList {
		objKey := client.ObjectKeyFromObject(bmh)
		r.Logger.Info("Applying BaremetalHost", "BMH", objKey)
		if err := applyRuntimeObject(ctx, objKey, bmh, r.Client); err != nil {
			return err
		}
	}
	return nil
}

func (r *BMHManager) UnScheduleVMs(ctx context.Context) error {
	podList, err := r.getPods(ctx)
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		k8sNode, err := r.getNode(ctx, pod)
		if err != nil {
			return err
		}
		annotations := k8sNode.GetAnnotations()
		if k8sNode.GetAnnotations() == nil {
			continue
		}

		delete(annotations, vinov1.VinoNodeNetworkValuesAnnotation)
		k8sNode.SetAnnotations(annotations)
		// TODO consider accumulating errors instead
		if err = r.Update(ctx, k8sNode); err != nil {
			return err
		}
	}
	return nil
}

func (r *BMHManager) getPods(ctx context.Context) (*corev1.PodList, error) {
	labelOpt := client.MatchingLabels{
		vinov1.VinoLabelDSNameSelector:      r.ViNO.Name,
		vinov1.VinoLabelDSNamespaceSelector: r.ViNO.Namespace,
	}

	nsOpt := client.InNamespace(r.Namespace)

	podList := &corev1.PodList{}
	return podList, r.List(ctx, podList, labelOpt, nsOpt)
}

// requestVMs iterates over each vino-builder pod, and annotates a k8s node for the pod
// with a request for VMs. Each vino-builder pod waits for the annotation.
// when annotation with VM request is added to a k8s node, vino manager WaitVMs should be used before creating BMHs
func (r *BMHManager) requestVMs(ctx context.Context) error {
	podList, err := r.getPods(ctx)
	if err != nil {
		return err
	}

	r.Logger.Info("Vino daemonset pod count", "count", len(podList.Items))

	for _, pod := range podList.Items {
		r.Logger.Info("Creating baremetal hosts for pod",
			"pod name",
			types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
		)
		err := r.createIpamNetworks(ctx, r.ViNO)
		if err != nil {
			return err
		}
		err = r.setBMHs(ctx, pod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *BMHManager) createIpamNetworks(ctx context.Context, vino *vinov1.Vino) error {
	// TODO (kkalynovskyi) these needs to be propagated into network template, and be configurable
	// TODO (kkalynovskyi) develop generic network templates that would allow to handle all networks
	// in single generic way.
	// Bootnetwork needs to be handled spearately because it needs to be created by libvirt
	// And have different configuration.
	if r.BootNetwork == nil {
		r.BootNetwork = &vinov1.Network{
			SubNet:          "10.153.241.0/24",
			AllocationStart: "10.153.241.2",
			AllocationStop:  "10.153.241.254",
			Name:            "pxe-boot",
			MACPrefix: "52:54:00:32:00:00",
		}
	}
	networks := vino.Spec.Networks
	// Append bootnetwork to be created in IPAM
	networks = append(networks, *r.BootNetwork)
	for _, network := range networks {
		if err := r.createIpamNetwork(ctx, network); err != nil {
			return err
		}
	}
	return nil
}

func (r *BMHManager) createIpamNetwork(ctx context.Context, network vinov1.Network) error {
	subnetRange, err := ipam.NewRange(network.AllocationStart, network.AllocationStop)
	if err != nil {
		return err
	}
	macPrefix := network.MACPrefix
	if macPrefix == "" {
		r.Logger.Info("No MACPrefix provided; using default MACPrefix for network",
			"default prefix", DefaultMACPrefix, "network name", network.Name)
		macPrefix = DefaultMACPrefix
	}
	return r.Ipam.AddSubnetRange(ctx, network.SubNet, subnetRange, macPrefix)
}

func (r *BMHManager) setBMHs(ctx context.Context, pod corev1.Pod) error {
	domains := []vinov1.BuilderDomain{}

	k8sNode, err := r.getNode(ctx, pod)
	if err != nil {
		return err
	}

	nodeNetworks, err := r.nodeNetworks(ctx, r.ViNO.Spec.Networks, k8sNode)
	if err != nil {
		return err
	}

	for _, node := range r.ViNO.Spec.Nodes {
		r.Logger.Info("Saving BMHs for vino node", "node name", node.Name, "count", node.Count)
		prefix := r.getBMHNodePrefix(pod)
		for i := 0; i < node.Count; i++ {
			roleSuffix := fmt.Sprintf("%s-%d", node.Name, i)
			bmhName := fmt.Sprintf("%s-%s", prefix, roleSuffix)

			domainValues, nodeErr := r.domainSpecificNetValues(ctx, bmhName, node, nodeNetworks)
			if nodeErr != nil {
				return nodeErr
			}
			domainValues.Name = roleSuffix
			domainValues.Role = node.Name

			// Append a specific domain to the list
			domains = append(domains, domainValues.BuilderDomain)

			netData, netDataNs, nodeErr := r.setBMHNetworkSecret(ctx, node, domainValues)
			if nodeErr != nil {
				return nodeErr
			}

			bmcAddr, labels, nodeErr := r.getBMCAddressAndLabels(k8sNode, roleSuffix)
			if nodeErr != nil {
				return nodeErr
			}

			for label, value := range node.BMHLabels {
				labels[label] = value
			}

			rootDeviceName := node.RootDeviceName
			if rootDeviceName == "" {
				rootDeviceName = vinov1.VinoDefaultRootDeviceName
			}

			credentialSecretName := r.setBMHCredentials(bmhName)
			bmh := &metal3.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bmhName,
					Namespace: r.Namespace,
					Labels:    labels,
				},
				Spec: metal3.BareMetalHostSpec{
					NetworkData: &corev1.SecretReference{
						Name:      netData,
						Namespace: netDataNs,
					},
					BMC: metal3.BMCDetails{
						Address:                        bmcAddr,
						CredentialsName:                credentialSecretName,
						DisableCertificateVerification: true,
					},
					BootMACAddress: domainValues.BootMACAddress,
					RootDeviceHints: &metal3.RootDeviceHints{
						DeviceName: rootDeviceName,
					},
				},
			}
			r.bmhList = append(r.bmhList, bmh)
		}
	}

	r.Logger.Info("annotating node", "node", k8sNode.Name)
	vinoBuilder := vinov1.Builder{
		PXEBootImageHost:     r.ViNO.Spec.PXEBootImageHost,
		PXEBootImageHostPort: r.ViNO.Spec.PXEBootImageHostPort,
		Networks:             r.ViNO.Spec.Networks,
		Nodes:                r.ViNO.Spec.Nodes,
		CPUConfiguration:     r.ViNO.Spec.CPUConfiguration,
		Domains:              domains,
	}
	return r.annotateNode(ctx, k8sNode, vinoBuilder)
}

// nodeNetworks returns a copy of node network with a unique per node values
func (r *BMHManager) nodeNetworks(ctx context.Context,
	globalNetworks []vinov1.Network,
	k8sNode *corev1.Node) ([]vinov1.Network, error) {
	for netIndex, network := range globalNetworks {
		for routeIndex, route := range network.Routes {
			if route.Gateway == "$vinobridge" {
				r.Logger.Info("Getting GW bridge IP from node", "node", k8sNode.Name)
				bridgeIP, err := r.getBridgeIP(ctx, k8sNode)
				if err != nil {
					return []vinov1.Network{}, err
				}
				globalNetworks[netIndex].Routes[routeIndex].Gateway = bridgeIP
			}
		}
	}
	return globalNetworks, nil
}

func (r *BMHManager) domainSpecificNetValues(
	ctx context.Context,
	bmhName string,
	node vinov1.NodeSet,
	networks []vinov1.Network) (networkTemplateValues, error) {
	// Allocate an IP for each of this BMH's network interfaces

	domainInterfaces := []vinov1.BuilderNetworkInterface{}
	for _, iface := range node.NetworkInterfaces {
		networkName := iface.NetworkName
		subnet := ""
		var err error
		subnetRange := vinov1.Range{}
		for _, network := range networks {
			if network.Name == networkName {
				subnet = network.SubNet
				subnetRange, err = ipam.NewRange(network.AllocationStart, network.AllocationStop)
				if err != nil {
					return networkTemplateValues{}, err
				}
				break
			}
		}
		if subnet == "" {
			return networkTemplateValues{}, fmt.Errorf("Interface %s doesn't have a matching network defined", networkName)
		}
		ipAllocatedTo := fmt.Sprintf("%s/%s", bmhName, iface.NetworkName)
		ipAddress, macAddress, err := r.Ipam.AllocateIP(ctx, subnet, subnetRange, ipAllocatedTo)
		if err != nil {
			return networkTemplateValues{}, err
		}
		domainInterfaces = append(domainInterfaces, vinov1.BuilderNetworkInterface{
			IPAddress:        ipAddress,
			MACAddress:       macAddress,
			NetworkInterface: iface,
		})

		r.Logger.Info("Got MAC and IP for the network and node",
			"MAC", macAddress, "IP", ipAddress, "bmh name", bmhName)
	}
	// Handle bootMAC separately
	bootMAC, err := r.generatePXEBootMAC(ctx, bmhName)
	if err != nil {
		return networkTemplateValues{}, err
	}
	r.Logger.Info("Got bootMAC address for BMH node", "bmh name", bmhName, "bootMAC", bootMAC)
	return networkTemplateValues{
		Node:     node,
		BMHName:  bmhName,
		Networks: networks,
		BuilderDomain: vinov1.BuilderDomain{
			BootMACAddress: bootMAC,
			Interfaces:     domainInterfaces,
		},
	}, nil
}

func (r *BMHManager) generatePXEBootMAC(ctx context.Context, bmhName string) (string, error) {
	subnetRange, err := ipam.NewRange(r.BootNetwork.AllocationStart, r.BootNetwork.AllocationStop)
	if err != nil {
		return "", err
	}

	ipAllocatedTo := fmt.Sprintf("%s/%s", bmhName, "pxe-boot")
	_, mac, err := r.Ipam.AllocateIP(ctx, r.BootNetwork.SubNet, subnetRange, ipAllocatedTo)
	return mac, err
}

func (r *BMHManager) annotateNode(ctx context.Context, k8sNode *corev1.Node, vinoBuilder vinov1.Builder) error {
	b, err := yaml.Marshal(vinoBuilder)
	if err != nil {
		return err
	}

	annotations := k8sNode.GetAnnotations()
	if k8sNode.GetAnnotations() == nil {
		annotations = make(map[string]string)
	}

	annotations[vinov1.VinoNodeNetworkValuesAnnotation] = string(b)
	k8sNode.SetAnnotations(annotations)

	return r.Update(ctx, k8sNode)
}

func (r *BMHManager) getBridgeIP(ctx context.Context, k8sNode *corev1.Node) (string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctxTimeout.Done():
			return "", ctx.Err()
		default:
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: k8sNode.Name,
				},
			}
			if err := r.Get(ctx, client.ObjectKeyFromObject(node), node); err != nil {
				return "", err
			}

			ip, exist := k8sNode.Labels[vinov1.VinoDefaultGatewayBridgeLabel]
			if exist {
				return ip, nil
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func (r *BMHManager) getNode(ctx context.Context, pod corev1.Pod) (*corev1.Node, error) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.Spec.NodeName,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(node), node)
	return node, err
}

func (r *BMHManager) getBMHNodePrefix(pod corev1.Pod) string {
	// TODO we need to do something about name length limitations
	return fmt.Sprintf("%s-%s-%s", r.ViNO.Namespace, r.ViNO.Name, pod.Spec.NodeName)
}

func (r *BMHManager) getBMCAddressAndLabels(
	node *corev1.Node,
	vmName string) (string, map[string]string, error) {
	logger := r.Logger.WithValues("k8s node", node.Name)
	labels := map[string]string{}
	for _, key := range r.ViNO.Spec.NodeLabelKeysToCopy {
		value, ok := node.Labels[key]
		if !ok {
			logger.Info("Kubernetes node missing label from vino CR CopyNodeLabelKeys field", "label", key)
		}
		labels[key] = value
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return fmt.Sprintf("redfish+http://%s:%d/redfish/v1/Systems/%s", addr.Address, 8000, vmName), labels, nil
		}
	}
	return "", labels, fmt.Errorf("Node %s doesn't have internal ip address defined", node.Name)
}

// setBMHCredentials returns secret name with credentials and error
func (r *BMHManager) setBMHCredentials(bmhName string) string {
	credName := fmt.Sprintf("%s-%s", bmhName, "credentials")
	bmhCredentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credName,
			Namespace: r.Namespace,
		},
		StringData: map[string]string{
			"username": r.ViNO.Spec.BMCCredentials.Username,
			"password": r.ViNO.Spec.BMCCredentials.Password,
		},
		Type: corev1.SecretTypeOpaque,
	}
	r.credentialSecrets = append(r.credentialSecrets, bmhCredentialSecret)
	return credName
}

func (r *BMHManager) setBMHNetworkSecret(
	ctx context.Context,
	node vinov1.NodeSet,
	values networkTemplateValues) (string, string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.NetworkDataTemplate.Name,
			Namespace: node.NetworkDataTemplate.Namespace,
		},
	}

	logger := r.Logger.WithValues("vino node", node.Name, "vino", client.ObjectKeyFromObject(r.ViNO))

	objKey := client.ObjectKeyFromObject(secret)
	logger.Info("Looking for secret with network template for vino node", "secret", objKey)
	if err := r.Get(ctx, objKey, secret); err != nil {
		return "", "", err
	}

	rawTmpl, ok := secret.Data[vinov1.VinoNetworkDataTemplateDefaultKey]
	if !ok {
		return "", "", fmt.Errorf("network template secret %v has no key '%s'",
			objKey,
			vinov1.VinoNetworkDataTemplateDefaultKey)
	}

	tpl, err := template.New("net-template").Funcs(sprig.TxtFuncMap()).Parse(string(rawTmpl))
	if err != nil {
		return "", "", err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tpl.Execute(buf, values)
	if err != nil {
		return "", "", err
	}

	name := fmt.Sprintf("%s-network-data", values.BMHName)
	r.networkSecrets = append(r.networkSecrets, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.Namespace,
		},
		StringData: map[string]string{
			"networkData": buf.String(),
		},
		Type: corev1.SecretTypeOpaque,
	})
	return name, r.Namespace, nil
}

func applyRuntimeObject(ctx context.Context, key client.ObjectKey, obj client.Object, c client.Client) error {
	getObj := obj
	err := c.Get(ctx, key, getObj)
	switch {
	case apierror.IsNotFound(err):
		err = c.Create(ctx, obj)
	case err == nil:
		err = c.Patch(ctx, obj, client.MergeFrom(getObj))
	}
	return err
}
