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

package controllers

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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerror "k8s.io/apimachinery/pkg/util/errors"
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
	Node      vinov1.NodeSet // the specific node type to be templated
	BMHName   string
	Networks  []vinov1.Network
	Generated generatedValues // Host-specific values calculated by ViNO: IP, etc
}

type generatedValues struct {
	IPAddresses  map[string]string
	MACAddresses map[string]string
}

func (r *VinoReconciler) ensureBMHs(ctx context.Context, vino *vinov1.Vino) error {
	labelOpt := client.MatchingLabels{
		vinov1.VinoLabelDSNameSelector:      vino.Name,
		vinov1.VinoLabelDSNamespaceSelector: vino.Namespace,
	}

	nsOpt := client.InNamespace(getRuntimeNamespace())

	podList := &corev1.PodList{}
	err := r.List(ctx, podList, labelOpt, nsOpt)
	if err != nil {
		return err
	}

	logger := logr.FromContext(ctx)
	logger.Info("Vino daemonset pod count", "count", len(podList.Items))

	for _, pod := range podList.Items {
		logger.Info("Creating baremetal hosts for pod",
			"pod name",
			types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
		)
		err := r.createIpamNetworks(ctx, vino)
		if err != nil {
			return err
		}
		err = r.createBMHperPod(ctx, vino, pod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *VinoReconciler) reconcileBMHs(ctx context.Context, vino *vinov1.Vino) error {
	if err := r.ensureBMHs(ctx, vino); err != nil {
		err = fmt.Errorf("could not reconcile BaremetalHosts: %w", err)
		apimeta.SetStatusCondition(&vino.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionFalse,
			Reason:             vinov1.ReconciliationFailedReason,
			Message:            err.Error(),
			Type:               vinov1.ConditionTypeReady,
			ObservedGeneration: vino.GetGeneration(),
		})
		apimeta.SetStatusCondition(&vino.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionFalse,
			Reason:             vinov1.ReconciliationFailedReason,
			Message:            err.Error(),
			Type:               vinov1.ConditionTypeBMHReady,
			ObservedGeneration: vino.GetGeneration(),
		})
		if patchStatusErr := r.patchStatus(ctx, vino); patchStatusErr != nil {
			err = kerror.NewAggregate([]error{err, patchStatusErr})
			err = fmt.Errorf("unable to patch status after BaremetalHosts reconciliation failed: %w", err)
		}
		return err
	}
	apimeta.SetStatusCondition(&vino.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             vinov1.ReconciliationSucceededReason,
		Message:            "BaremetalHosts reconciled",
		Type:               vinov1.ConditionTypeBMHReady,
		ObservedGeneration: vino.GetGeneration(),
	})
	if err := r.patchStatus(ctx, vino); err != nil {
		err = fmt.Errorf("unable to patch status after BaremetalHosts reconciliation succeeded: %w", err)
		return err
	}
	return nil
}

func (r *VinoReconciler) createIpamNetworks(ctx context.Context, vino *vinov1.Vino) error {
	logger := logr.FromContext(ctx)
	for _, network := range vino.Spec.Networks {
		subnetRange, err := ipam.NewRange(network.AllocationStart, network.AllocationStop)
		if err != nil {
			return err
		}
		macPrefix := network.MACPrefix
		if macPrefix == "" {
			logger.Info("No MACPrefix provided; using default MACPrefix for network",
				"default prefix", DefaultMACPrefix, "network name", network.Name)
			macPrefix = DefaultMACPrefix
		}
		err = r.Ipam.AddSubnetRange(ctx, network.SubNet, subnetRange, macPrefix)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *VinoReconciler) createBMHperPod(ctx context.Context, vino *vinov1.Vino, pod corev1.Pod) error {
	logger := logr.FromContext(ctx)

	nodeNetworkValues := map[string]generatedValues{}

	k8sNode, err := r.getNode(ctx, pod)
	if err != nil {
		return err
	}

	ip, err := r.getBridgeIP(ctx, k8sNode)
	if err != nil {
		return err
	}

	for _, node := range vino.Spec.Nodes {
		logger.Info("Creating BMHs for vino node", "node name", node.Name, "count", node.Count)
		prefix := r.getBMHNodePrefix(vino, pod)
		for i := 0; i < node.Count; i++ {
			roleSuffix := fmt.Sprintf("%s-%d", node.Name, i)
			bmhName := fmt.Sprintf("%s-%s", prefix, roleSuffix)

			creds, nodeErr := r.reconcileBMHCredentials(ctx, vino)
			if nodeErr != nil {
				return nodeErr
			}

			values, nodeErr := r.networkValues(ctx, bmhName, ip, node, vino)
			if nodeErr != nil {
				return nodeErr
			}
			nodeNetworkValues[roleSuffix] = values.Generated

			netData, netDataNs, nodeErr := r.reconcileBMHNetworkData(ctx, node, vino, values)
			if nodeErr != nil {
				return nodeErr
			}

			bmcAddr, labels, nodeErr := r.getBMCAddressAndLabels(ctx, k8sNode, vino.Spec.NodeLabelKeysToCopy, roleSuffix)
			if nodeErr != nil {
				return nodeErr
			}

			for label, value := range node.BMHLabels {
				labels[label] = value
			}

			bmh := &metal3.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bmhName,
					Namespace: getRuntimeNamespace(),
					// TODO add rack and server labels, when we crearly define mechanism
					// which labels we are copying
					Labels: labels,
				},
				Spec: metal3.BareMetalHostSpec{
					NetworkData: &corev1.SecretReference{
						Name:      netData,
						Namespace: netDataNs,
					},
					BMC: metal3.BMCDetails{
						Address:                        bmcAddr,
						CredentialsName:                creds,
						DisableCertificateVerification: true,
					},
				},
			}
			objKey := client.ObjectKeyFromObject(bmh)
			logger.Info("Creating BMH", "name", objKey)
			nodeErr = applyRuntimeObject(ctx, objKey, bmh, r.Client)
			if nodeErr != nil {
				return nodeErr
			}
		}
	}
	logger.Info("annotating node", "node", k8sNode.Name)
	if err = r.annotateNode(ctx, ip, k8sNode, nodeNetworkValues); err != nil {
		return err
	}
	return nil
}

func (r *VinoReconciler) networkValues(
	ctx context.Context,
	bmhName string,
	bridgeIP string,
	node vinov1.NodeSet,
	vino *vinov1.Vino) (networkTemplateValues, error) {
	// Allocate an IP for each of this BMH's network interfaces
	ipAddresses := map[string]string{}
	macAddresses := map[string]string{}
	for _, iface := range node.NetworkInterfaces {
		networkName := iface.NetworkName
		subnet := ""
		var err error
		subnetRange := vinov1.Range{}
		for netIndex, network := range vino.Spec.Networks {
			for routeIndex, route := range network.Routes {
				if route.Gateway == "$vinobridge" {
					vino.Spec.Networks[netIndex].Routes[routeIndex].Gateway = bridgeIP
				}
			}
			if network.Name == networkName {
				subnet = network.SubNet
				subnetRange, err = ipam.NewRange(network.AllocationStart,
					network.AllocationStop)
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
		ipAddresses[networkName] = ipAddress
		macAddresses[iface.Name] = macAddress
		logr.FromContext(ctx).Info("Got MAC and IP for the network and node",
			"MAC", macAddress, "IP", ipAddress, "bmh name", bmhName)
	}
	return networkTemplateValues{
		Node:     node,
		BMHName:  bmhName,
		Networks: vino.Spec.Networks,
		Generated: generatedValues{
			IPAddresses:  ipAddresses,
			MACAddresses: macAddresses,
		},
	}, nil
}

func (r *VinoReconciler) annotateNode(ctx context.Context,
	gwIP string,
	k8sNode *corev1.Node,
	values map[string]generatedValues) error {
	logr.FromContext(ctx).Info("Getting GW bridge IP from node", "node", k8sNode.Name)
	builderValues := vinov1.Builder{
		Domains:    make(map[string]vinov1.BuilderDomain),
		GWIPBridge: gwIP,
	}
	for domainName, domain := range values {
		builderDomain := vinov1.BuilderDomain{
			Interfaces: make(map[string]vinov1.BuilderNetworkInterface),
		}
		for ifName, ifMAC := range domain.MACAddresses {
			builderDomain.Interfaces[ifName] = vinov1.BuilderNetworkInterface{
				MACAddress: ifMAC,
			}
		}
		builderValues.Domains[domainName] = builderDomain
	}

	b, err := yaml.Marshal(builderValues)
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

func (r *VinoReconciler) getBridgeIP(ctx context.Context, k8sNode *corev1.Node) (string, error) {
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

func (r *VinoReconciler) getNode(ctx context.Context, pod corev1.Pod) (*corev1.Node, error) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.Spec.NodeName,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(node), node)
	return node, err
}

func (r *VinoReconciler) getBMHNodePrefix(vino *vinov1.Vino, pod corev1.Pod) string {
	// TODO we need to do something about name length limitations
	return fmt.Sprintf("%s-%s-%s", vino.Namespace, vino.Name, pod.Spec.NodeName)
}

func (r *VinoReconciler) getBMCAddressAndLabels(
	ctx context.Context,
	node *corev1.Node,
	labelKeys []string,
	vmName string) (string, map[string]string, error) {
	logger := logr.FromContext(ctx).WithValues("k8s node", node.Name)

	labels := map[string]string{}

	for _, key := range labelKeys {
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

// reconcileBMHCredentials returns secret name with credentials and error
func (r *VinoReconciler) reconcileBMHCredentials(ctx context.Context, vino *vinov1.Vino) (string, error) {
	ns := getRuntimeNamespace()
	// coresponds to DS name, since we have only one DS per vino CR
	credentialSecretName := fmt.Sprintf("%s-%s", r.getDaemonSetName(vino), "credentials")
	netSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialSecretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			"username": vino.Spec.BMCCredentials.Username,
			"password": vino.Spec.BMCCredentials.Password,
		},
		Type: corev1.SecretTypeOpaque,
	}

	objKey := client.ObjectKeyFromObject(netSecret)

	if err := applyRuntimeObject(ctx, objKey, netSecret, r.Client); err != nil {
		return "", err
	}
	return credentialSecretName, nil
}

func (r *VinoReconciler) reconcileBMHNetworkData(
	ctx context.Context,
	node vinov1.NodeSet,
	vino *vinov1.Vino,
	values networkTemplateValues) (string, string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.NetworkDataTemplate.Name,
			Namespace: node.NetworkDataTemplate.Namespace,
		},
	}

	logger := logr.FromContext(ctx).WithValues("vino node", node.Name, "vino", client.ObjectKeyFromObject(vino))

	objKey := client.ObjectKeyFromObject(secret)
	logger.Info("Looking for secret with network template for vino node", "secret", objKey)
	if err := r.Get(ctx, objKey, secret); err != nil {
		return "", "", err
	}

	rawTmpl, ok := secret.Data[TemplateDefaultKey]
	if !ok {
		return "", "", fmt.Errorf("network template secret %v has no key '%s'", objKey, TemplateDefaultKey)
	}

	tpl, err := template.New("net-template").Funcs(sprig.TxtFuncMap()).Parse(string(rawTmpl))
	if err != nil {
		return "", "", err
	}

	logger.Info("Genereated MAC Addresses values are", "GENERATED VALUES", values.Generated.MACAddresses)

	buf := bytes.NewBuffer([]byte{})
	err = tpl.Execute(buf, values)
	if err != nil {
		return "", "", err
	}

	name := fmt.Sprintf("%s-network-data", values.BMHName)
	ns := getRuntimeNamespace()
	netSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		StringData: map[string]string{
			"networkData": buf.String(),
		},
		Type: corev1.SecretTypeOpaque,
	}

	objKey = client.ObjectKeyFromObject(netSecret)

	logger.Info("Creating network secret for vino node", "secret", objKey)

	if err := applyRuntimeObject(ctx, objKey, netSecret, r.Client); err != nil {
		return "", "", err
	}
	return name, ns, nil
}
