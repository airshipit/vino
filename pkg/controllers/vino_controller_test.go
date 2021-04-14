package controllers

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vinov1 "vino/pkg/api/v1"
)

func testDS() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{Spec: appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{}}}}}
}

func testVINO() *vinov1.Vino {
	return &vinov1.Vino{
		ObjectMeta: v1.ObjectMeta{
			Name:      "vino",
			Namespace: "default",
		},
		Spec: vinov1.VinoSpec{
			Networks: []vinov1.Network{}}}
}

var _ = Describe("Test Setting Env variables", func() {
	Context("when daemonset is created", func() {
		l := logr.Discard()
		ctx := logr.NewContext(context.Background(), l)
		Context("when daemonset has containers", func() {
			It("sets env variable to every container", func() {
				ifName := "eth0"
				ds := testDS()
				ds.Spec.Template.Spec.Containers = make([]corev1.Container, 3)

				setEnv(ctx, ds, vinov1.EnvVarVMInterfaceName, ifName)

				for _, container := range ds.Spec.Template.Spec.Containers {
					Expect(container.Env).To(HaveLen(1))
					Expect(container.Env[0].Name).To(Equal(vinov1.EnvVarVMInterfaceName))
					Expect(container.Env[0].Value).To(Equal(ifName))
				}
			})

		})
		Context("when daemonset has container with pre-existing env var values", func() {
			It("overrides that variable in the container", func() {
				ifName := "eth0"
				ds := testDS()
				ds.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  vinov1.EnvVarVMInterfaceName,
								Value: "old-value",
							},
						},
					},
				}

				setEnv(ctx, ds, vinov1.EnvVarVMInterfaceName, ifName)
				Expect(ds.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := ds.Spec.Template.Spec.Containers[0]
				Expect(container.Env).To(HaveLen(1))
				Expect(container.Env[0].Name).To(Equal(vinov1.EnvVarVMInterfaceName))
				Expect(container.Env[0].Value).To(Equal(ifName))
			})
		})
		Context("when daemonset containers don't have required variable", func() {
			It("adds that variable to all the containers", func() {
				ifName := "eth0"
				ds := testDS()
				ds.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "bar",
								Value: "old-value",
							},
							{
								Name:  "foo",
								Value: "old-value",
							},
						},
					},
					{
						Env: []corev1.EnvVar{
							{
								Name:  "foo",
								Value: "old-value",
							},
							{
								Name:  "bar",
								Value: "old-value",
							},
						},
					},
				}

				setEnv(ctx, ds, vinov1.EnvVarVMInterfaceName, ifName)

				Expect(ds.Spec.Template.Spec.Containers).To(HaveLen(2))
				for _, container := range ds.Spec.Template.Spec.Containers {
					Expect(container.Env).To(HaveLen(3))
					for i, env := range container.Env {
						if i == len(container.Env)-1 {
							// only last env holds the correct values
							Expect(env.Name).To(Equal(vinov1.EnvVarVMInterfaceName))
							Expect(env.Value).To(Equal(ifName))
						} else {
							Expect(env.Name).NotTo(Equal(vinov1.EnvVarVMInterfaceName))
							Expect(env.Value).NotTo(Equal(ifName))
						}
					}
				}

			})
		})
		Context("when daemonset container has many variables", func() {
			It("it sets required variable only single time", func() {
				ifName := "eth0"
				ds := testDS()
				ds.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "foo",
								Value: "old-value",
							},
							{
								Name:  vinov1.EnvVarVMInterfaceName,
								Value: "old-value",
							},
							{
								Name:  "bar",
								Value: "old-value",
							},
						},
					},
				}

				setEnv(ctx, ds, vinov1.EnvVarVMInterfaceName, ifName)
				Expect(ds.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := ds.Spec.Template.Spec.Containers[0]
				Expect(container.Env).To(HaveLen(3))
				for i, env := range container.Env {
					if i == 1 {
						// only env var with index 1 holds the correct values
						Expect(env.Name).To(Equal(vinov1.EnvVarVMInterfaceName))
						Expect(env.Value).To(Equal(ifName))
					} else {
						Expect(env.Name).NotTo(Equal(vinov1.EnvVarVMInterfaceName))
						Expect(env.Value).NotTo(Equal(ifName))
					}
				}
			})
		})
	})
})
