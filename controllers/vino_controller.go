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
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerror "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	vinov1 "vino/api/v1"
)

const (
	DaemonSetTemplateDefaultDataKey = "template"

	ContainerNameLibvirt = "libvirt"
	ConfigMapKeyVinoSpec = "vino-spec"

	// TODO (kkalynovskyi) remove this, when moving to default libvirt template.
	DefaultImageLibvirt = "quay.io/teoyaomiqui/libvirt"
)

// VinoReconciler reconciles a Vino object
type VinoReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes/status,verbs=get;update;patch

func (r *VinoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("vino", req.NamespacedName)

	ctx = context.TODO()
	vino := &vinov1.Vino{}
	if err := r.Get(ctx, req.NamespacedName, vino); err != nil {
		if apierror.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get vino CR")
		return ctrl.Result{}, err
	}

	var err error
	if vino.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(vino, vinov1.VinoFinalizer) {
		logger.Info("adding finalizer to new vino object")
		controllerutil.AddFinalizer(vino, vinov1.VinoFinalizer)
		err = r.Update(ctx, vino)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	defer func() {
		if dErr := r.Status().Update(ctx, vino); dErr != nil {
			if err != nil {
				err = kerror.NewAggregate([]error{err, dErr})
			}
			err = dErr
		}
		logger.Info("updated vino CR status")
	}()

	if !vino.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.finalize(ctx, vino)
	}

	err = r.ensureConfigMap(ctx, req.NamespacedName, vino)
	if err != nil {
		readyCondition := vinov1.Condition{
			Status:  corev1.ConditionFalse,
			Reason:  "Error has occurred while making sure that ConfigMap for VINO is in correct state",
			Message: err.Error(),
			Type:    vinov1.ConditionTypeReady,
		}
		r.setCondition(vino, readyCondition)
		vino.Status.ConfigMapReady = false
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 60}, err
	}
	vino.Status.ConfigMapReady = true

	err = r.ensureDaemonSet(ctx, req.NamespacedName, vino)
	if err != nil {
		readyCondition := vinov1.Condition{
			Status:  corev1.ConditionFalse,
			Reason:  "Error has occurred while making sure that VINO Daemonset is installed on kubernetes nodes",
			Message: err.Error(),
			Type:    vinov1.ConditionTypeReady,
		}
		r.setCondition(vino, readyCondition)
		vino.Status.DaemonSetReady = false
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 60}, err
	}
	vino.Status.DaemonSetReady = true

	r.setReadyStatus(vino)
	logger.Info("successfully reconciled VINO CR")
	return ctrl.Result{}, nil
}

func (r *VinoReconciler) ensureConfigMap(ctx context.Context, name types.NamespacedName, vino *vinov1.Vino) error {

	logger := r.Log.WithValues("vino", name)

	generatedCM, err := r.buildConfigMap(name, vino)
	if err != nil {
		return err
	}
	logger.Info("successfully built config map", "new config map data", generatedCM.Data)

	currentCM, err := r.getCurrentConfigMap(ctx, vino)
	if err != nil {
		return err
	}

	if currentCM == nil {
		logger.Info("current config map is not present in a cluster creating newly generated one")
		return applyRuntimeObject(
			ctx,
			types.NamespacedName{Name: generatedCM.Name, Namespace: generatedCM.Namespace},
			generatedCM,
			r.Client)
	}

	logger.Info("generated config map", "current config map data", currentCM.Data)

	if needsUpdate(generatedCM, currentCM) {
		logger.Info("current config map needs an update, trying to update it")
		return r.Client.Update(ctx, generatedCM)
	}
	return nil
}

func (r *VinoReconciler) buildConfigMap(name types.NamespacedName, vino *vinov1.Vino) (*corev1.ConfigMap, error) {
	r.Log.Info("Generating new config map for vino object")

	data, err := yaml.Marshal(vino.Spec)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Data: map[string]string{
			ConfigMapKeyVinoSpec: string(data),
		},
	}, nil
}

func (r *VinoReconciler) getCurrentConfigMap(ctx context.Context, vino *vinov1.Vino) (*corev1.ConfigMap, error) {
	r.Log.Info("Getting current config map for vino object")
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      vino.Name,
		Namespace: vino.Namespace,
	}, cm)
	if err != nil {
		if !apierror.IsNotFound(err) {
			return cm, err
		}
		return nil, nil
	}

	return cm, nil
}

func (r *VinoReconciler) setReadyStatus(vino *vinov1.Vino) {
	if vino.Status.ConfigMapReady && vino.Status.DaemonSetReady {
		r.Log.Info("All VINO components are in ready state, setting VINO CR to ready state")
		readyCondition := vinov1.Condition{
			Status:  corev1.ConditionTrue,
			Reason:  "Networking, Virtual Machines, DaemonSet and ConfigMap is in ready state",
			Message: "All VINO components are in ready state, setting VINO CR to ready state",
			Type:    vinov1.ConditionTypeReady,
		}
		r.setCondition(vino, readyCondition)
	}
}

func needsUpdate(generated, current *corev1.ConfigMap) bool {
	for key, value := range generated.Data {
		if current.Data[key] != value {
			return true
		}
	}
	return false
}

func (r *VinoReconciler) ensureDaemonSet(ctx context.Context, name types.NamespacedName, vino *vinov1.Vino) error {

	ds, err := r.overrideDaemonSet(ctx, vino)
	if err != nil {
		return err
	}

	if ds == nil {
		ds = defaultDaemonSet(vino)
	}

	r.decorateDaemonSet(ds, vino)

	if err := applyRuntimeObject(ctx, types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, ds, r.Client); err != nil {
		return err
	}

	// TODO (kkalynovskyi) this function needs to add owner reference on the daemonset set and watch
	// controller should watch for changes in daemonset to reconcile if it breaks, and change status
	// of the vino object
	// controlleruti.SetControllerReference(vino, ds, r.scheme)

	return r.waitDaemonSet(30, ds)
}

func (r *VinoReconciler) decorateDaemonSet(ds *appsv1.DaemonSet, vino *vinov1.Vino) {
	volume := "vino-spec"

	ds.Spec.Template.Spec.NodeSelector = vino.Spec.NodeSelector.MatchLabels
	ds.Name = vino.Name
	ds.Namespace = vino.Namespace

	found := false
	for _, vol := range ds.Spec.Template.Spec.Volumes {
		if vol.Name == "vino-spec" {
			found = true
			break
		}
	}
	if !found {
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: vino.Name},
				},
			},
		})
	}

	// add vino spec to each container
	for i, c := range ds.Spec.Template.Spec.Containers {
		found = false
		for _, mount := range c.VolumeMounts {
			if mount.Name == volume {
				found = true
			}
		}
		if !found {
			r.Log.Info("volume mount with vino spec is not found",
				"vino instance", vino.Namespace+"/"+vino.Name,
				"container name", c.Name,
			)
			ds.Spec.Template.Spec.Containers[i].VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				MountPath: "/vino/spec",
				Name:      volume,
				ReadOnly:  true,
				SubPath:   ConfigMapKeyVinoSpec,
			})
		}
	}
}

func (r *VinoReconciler) waitDaemonSet(timeout int, ds *appsv1.DaemonSet) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	logger := r.Log.WithValues(
		"timeout in seconds", timeout,
		"daemonset", ds.Namespace+"/"+ds.Name)
	for {
		select {
		case <-ctx.Done():
			logger.Info("context canceled")
			return ctx.Err()
		default:
			getDS := &appsv1.DaemonSet{}
			err := r.Get(ctx, types.NamespacedName{
				Name:      ds.Name,
				Namespace: ds.Namespace,
			}, getDS)
			if err != nil {
				logger.Info("received error while waiting for ds to become ready, sleeping",
					"error", err.Error())
			} else {
				logger.Info("checking daemonset status", "status", getDS.Status)
				if getDS.Status.DesiredNumberScheduled == getDS.Status.NumberReady &&
					getDS.Status.DesiredNumberScheduled != 0 {
					logger.Info("daemonset is in ready status")
					return nil
				}
				logger.Info("daemonset is not in ready status, rechecking in 2 seconds")
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *VinoReconciler) overrideDaemonSet(ctx context.Context, vino *vinov1.Vino) (*appsv1.DaemonSet, error) {
	dsTemplate := vino.Spec.DaemonSetOptions.Template
	logger := r.Log.WithValues("DaemonSetTemplate", dsTemplate)
	cm := &corev1.ConfigMap{}

	if dsTemplate.Name == "" || dsTemplate.Namespace == "" {
		logger.Info("user provided vino DaemonSet template is empty or missing name or namespace")
		return nil, nil
	}

	err := r.Get(ctx, types.NamespacedName{
		Name:      dsTemplate.Name,
		Namespace: dsTemplate.Namespace,
	}, cm)
	if err != nil {
		// TODO check error if it doesn't exist, we should requeue request and wait for the template instead
		logger.Info("failed to get DaemonSet template does not exist in cluster", "error", err.Error())
		return nil, err
	}

	template, exist := cm.Data[DaemonSetTemplateDefaultDataKey]
	if !exist {
		logger.Info("malformed template provided data doesn't have key " + DaemonSetTemplateDefaultDataKey)
		return nil, fmt.Errorf("malformed template provided data doesn't have key " + DaemonSetTemplateDefaultDataKey)
	}

	ds := &appsv1.DaemonSet{}
	err = yaml.Unmarshal([]byte(template), ds)
	if err != nil {
		logger.Info("failed to unmarshal daemonset template", "error", err.Error())
		return nil, err
	}

	return ds, nil
}

func (r *VinoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vinov1.Vino{}).
		Complete(r)
}

func (r *VinoReconciler) finalize(ctx context.Context, vino *vinov1.Vino) error {
	// TODO aggregate errors instead
	if err := r.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: vino.Name, Namespace: vino.Namespace}}); err != nil {
		return err
	}
	if err := r.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: vino.Name, Namespace: vino.Namespace}}); err != nil {
		return err
	}
	controllerutil.RemoveFinalizer(vino, vinov1.VinoFinalizer)
	return r.Update(ctx, vino)
}

func (r *VinoReconciler) setCondition(vino *vinov1.Vino, condition vinov1.Condition) {
	for i, checkCondition := range vino.Status.Conditions {
		if checkCondition.Type == condition.Type {
			vino.Status.Conditions[i] = condition
			return
		}
	}
	vino.Status.Conditions = append([]vinov1.Condition{condition}, vino.Status.Conditions...)
}

func defaultDaemonSet(vino *vinov1.Vino) (ds *appsv1.DaemonSet) {
	libvirtImage := DefaultImageLibvirt

	if vino.Spec.DaemonSetOptions.LibvirtImage != "" {
		libvirtImage = vino.Spec.DaemonSetOptions.LibvirtImage
	}

	biDirectional := corev1.MountPropagationBidirectional

	ds = &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vino.Name,
			Namespace: vino.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"vino-id": vino.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"vino-id": vino.Name,
					},
				},
				Spec: corev1.PodSpec{
					HostPID:     true,
					HostNetwork: true,
					HostIPC:     true,
					Volumes: []corev1.Volume{
						{
							Name: "cgroup",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/fs/cgroup",
								},
							},
						},
						{
							Name: "dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
								},
							},
						},
						{
							Name: "run",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run",
								},
							},
						},
						{
							Name: "var-lib-libvirt",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/libvirt",
								},
							},
						},
						{
							Name: "var-lib-libvirt-images",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "var-lib-libvirt-images",
								},
							},
						},
						{
							Name: "logs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/libvirt",
								},
							},
						},
						{
							Name: "libmodules",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: ContainerNameLibvirt,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &[]bool{true}[0],
								RunAsUser:  &[]int64{0}[0],
							},
							Image:   libvirtImage,
							Command: []string{"/tmp/libvirt.sh"},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/lib/modules",
									Name:      "libmodules",
									ReadOnly:  true,
								},
								{
									MountPath:        "/var/lib/libvirt",
									Name:             "var-lib-libvirt",
									MountPropagation: &biDirectional,
								},
								{
									MountPath: "/var/lib/libvirt/images",
									Name:      "var-lib-libvirt-images",
								},
								{
									MountPath: "/run",
									Name:      "run",
								},
								{
									MountPath: "/dev",
									Name:      "dev",
								},
								{
									MountPath: "/sys/fs/cgroup",
									Name:      "cgroup",
								},
								{
									MountPath: "/var/log/libvirt",
									Name:      "logs",
								},
							},
						},
					},
				},
			},
		},
	}
	return
}

func applyRuntimeObject(ctx context.Context, key client.ObjectKey, obj client.Object, c client.Client) error {
	getObj := obj
	switch err := c.Get(ctx, key, getObj); {
	case apierror.IsNotFound(err):
		return c.Create(ctx, obj)
	case err == nil:
		return c.Update(ctx, obj)
	default:
		return err
	}
}
