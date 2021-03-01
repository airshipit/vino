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

	"github.com/go-logr/logr"
	metal3 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerror "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vinov1 "vino/pkg/api/v1"
)

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
		err := r.createBMHperPod(ctx, vino, pod)
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

func (r *VinoReconciler) createBMHperPod(ctx context.Context, vino *vinov1.Vino, pod corev1.Pod) error {
	for _, node := range vino.Spec.Node {
		logger := logr.FromContext(ctx)
		logger.Info("Creating BMHs for vino node", "node name", node.Name, "count", node.Count)
		prefix := r.getBMHNodePrefix(vino, pod)
		for i := 0; i < node.Count; i++ {
			roleSuffix := fmt.Sprintf("%s-%d", node.Name, i)
			bmhName := fmt.Sprintf("%s-%s", prefix, roleSuffix)

			creds, err := r.reconcileBMHCredentials(ctx, vino)
			if err != nil {
				return err
			}

			netData, netDataNs, err := r.reconcileBMHNetworkData(ctx, node, vino, nil)
			if err != nil {
				return err
			}

			bmcAddr, err := r.getBMCAddress(ctx, pod, roleSuffix)
			if err != nil {
				return err
			}

			bmh := &metal3.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bmhName,
					Namespace: getRuntimeNamespace(),
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
			err = applyRuntimeObject(ctx, objKey, bmh, r.Client)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *VinoReconciler) getBMHNodePrefix(vino *vinov1.Vino, pod corev1.Pod) string {
	// TODO we need to do something about name length limitations
	return fmt.Sprintf("%s-%s-%s", vino.Namespace, vino.Name, pod.Spec.NodeName)
}

func (r *VinoReconciler) getBMCAddress(
	ctx context.Context,
	pod corev1.Pod,
	vmName string) (string, error) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.Spec.NodeName,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(node), node)
	if err != nil {
		return "", err
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return fmt.Sprintf("redfish+http://%s:%d/redfish/v1/Systems/%s", addr.Address, 8000, vmName), nil
		}
	}
	return "", fmt.Errorf("Node %s doesn't have internal ip address defined", node.Name)
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
	values interface{}) (string, string, error) {
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

	tpl, err := template.New("net-template").Parse(string(rawTmpl))
	if err != nil {
		return "", "", err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tpl.Execute(buf, values)
	if err != nil {
		return "", "", err
	}

	name := fmt.Sprintf("%s-%s-%s", vino.Namespace, vino.Name, node.Name)

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
