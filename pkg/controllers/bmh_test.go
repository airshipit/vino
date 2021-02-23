package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	metal3 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	vinov1 "vino/pkg/api/v1"
)

// TODO expand tests when network and credential secret support is implemented
var _ = Describe("Test BMH reconciliation", func() {
	Context("when there are 2 k8s pods and worker count is 3", func() {
		It("creates 6 BMH hosts", func() {
			os.Setenv("RUNTIME_NAMESPACE", "vino-system")
			defer os.Unsetenv("RUNTIME_NAMESPACE")
			vino := testVINO()
			vino.Spec.Node = []vinov1.NodeSet{
				{
					Name:  "worker",
					Count: 3,
				},
			}

			podList := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "node01-pod",
							Namespace: "vino-system",
							Labels: map[string]string{
								vinov1.VinoLabelDSNameSelector:      vino.Name,
								vinov1.VinoLabelDSNamespaceSelector: vino.Namespace,
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "node01",
						},
					},

					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "node02-pod",
							Namespace: "vino-system",
							Labels: map[string]string{
								vinov1.VinoLabelDSNameSelector:      vino.Name,
								vinov1.VinoLabelDSNamespaceSelector: vino.Namespace,
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "node02",
						},
					},
				},
			}

			node1 := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node01",
				},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeInternalIP,
							Address: "10.0.0.2",
						},
					},
				},
			}
			node2 := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node02",
				},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeInternalIP,
							Address: "10.0.0.1",
						},
					},
				},
			}

			fake.NewClientBuilder()
			reconciler := &VinoReconciler{
				Client: fake.NewFakeClient(podList, node1, node2, vino),
			}

			l := zap.New(zap.UseDevMode(true))
			ctx := logr.NewContext(context.Background(), l)

			Expect(reconciler.reconcileBMHs(ctx, vino)).Should(Succeed())
			bmhName := "default-vino-node01-worker-1"

			bmh := &metal3.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bmhName,
					Namespace: "vino-system",
				},
			}
			Expect(reconciler.Get(ctx, client.ObjectKeyFromObject(bmh), bmh)).Should(Succeed())
			Expect(bmh.Spec.BMC.Address).To(Equal("redfish+http://10.0.0.2:8000/redfish/v1/Systems/worker-1"))
		})
	})
})
