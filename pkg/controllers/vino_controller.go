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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerror "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	vinov1 "vino/pkg/api/v1"
)

const (
	DaemonSetTemplateDefaultDataKey   = "template"
	DaemonSetTemplateDefaultName      = "vino-daemonset-template"
	DaemonSetTemplateDefaultNamespace = "vino-system"

	ContainerNameLibvirt = "libvirt"
	ConfigMapKeyVinoSpec = "vino-spec"
)

// VinoReconciler reconciles a Vino object
type VinoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes/status,verbs=get;update;patch

func (r *VinoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logr.FromContext(ctx)
	vino := &vinov1.Vino{}
	var err error

	if err = r.Get(ctx, req.NamespacedName, vino); err != nil {
		if apierror.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		err = fmt.Errorf("failed to get vino CR: %w", err)
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(vino, vinov1.VinoFinalizer) {
		logger.Info("adding finalizer to new vino object")
		controllerutil.AddFinalizer(vino, vinov1.VinoFinalizer)
		if err = r.Update(ctx, vino); err != nil {
			err = fmt.Errorf("unable to register finalizer: %w", err)
			return ctrl.Result{}, err
		}
	}

	if !vino.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.finalize(ctx, vino)
	}

	readyCondition := apimeta.FindStatusCondition(vino.Status.Conditions, vinov1.ConditionTypeReady)
	if readyCondition == nil || readyCondition.ObservedGeneration != vino.GetGeneration() {
		vinov1.VinoProgressing(vino)
		if err = r.patchStatus(ctx, vino); err != nil {
			err = fmt.Errorf("unable to patch status after progressing: %w", err)
			return ctrl.Result{Requeue: true}, err
		}
	}

	err = r.reconcileConfigMap(ctx, req.NamespacedName, vino)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	err = r.reconcileDaemonSet(ctx, vino)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	vinov1.VinoReady(vino)
	if err := r.patchStatus(ctx, vino); err != nil {
		err = fmt.Errorf("unable to patch status after reconciliation: %w", err)
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("successfully reconciled VINO CR")
	return ctrl.Result{}, nil
}

func (r *VinoReconciler) reconcileConfigMap(ctx context.Context, name types.NamespacedName, vino *vinov1.Vino) error {
	err := r.ensureConfigMap(ctx, name, vino)
	if err != nil {
		err = fmt.Errorf("could not reconcile ConfigMap: %w", err)
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
			Type:               vinov1.ConditionTypeConfigMapReady,
			ObservedGeneration: vino.GetGeneration(),
		})
		if patchStatusErr := r.patchStatus(ctx, vino); patchStatusErr != nil {
			err = kerror.NewAggregate([]error{err, patchStatusErr})
			err = fmt.Errorf("unable to patch status after ConfigMap reconciliation failed: %w", err)
		}
		return err
	}
	apimeta.SetStatusCondition(&vino.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             vinov1.ReconciliationSucceededReason,
		Message:            "ConfigMap reconciled",
		Type:               vinov1.ConditionTypeConfigMapReady,
		ObservedGeneration: vino.GetGeneration(),
	})
	if err = r.patchStatus(ctx, vino); err != nil {
		err = fmt.Errorf("unable to patch status after ConfigMap reconciliation succeeded: %w", err)
		return err
	}

	return nil
}

func (r *VinoReconciler) ensureConfigMap(ctx context.Context, name types.NamespacedName, vino *vinov1.Vino) error {
	logger := logr.FromContext(ctx)

	generatedCM, err := r.buildConfigMap(ctx, name, vino)
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

func (r *VinoReconciler) buildConfigMap(ctx context.Context, name types.NamespacedName, vino *vinov1.Vino) (
	*corev1.ConfigMap, error) {
	logr.FromContext(ctx).Info("Generating new config map for vino object")

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
	logr.FromContext(ctx).Info("Getting current config map for vino object")
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

func (r *VinoReconciler) patchStatus(ctx context.Context, vino *vinov1.Vino) error {
	key := client.ObjectKeyFromObject(vino)
	latest := &vinov1.Vino{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}
	return r.Client.Status().Patch(ctx, vino, client.MergeFrom(latest))
}

func needsUpdate(generated, current *corev1.ConfigMap) bool {
	for key, value := range generated.Data {
		if current.Data[key] != value {
			return true
		}
	}
	return false
}

func (r *VinoReconciler) reconcileDaemonSet(ctx context.Context, vino *vinov1.Vino) error {
	err := r.ensureDaemonSet(ctx, vino)
	if err != nil {
		err = fmt.Errorf("could not reconcile DaemonSet: %w", err)
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
			Type:               vinov1.ConditionTypeDaemonSetReady,
			ObservedGeneration: vino.GetGeneration(),
		})
		if patchStatusErr := r.patchStatus(ctx, vino); patchStatusErr != nil {
			err = kerror.NewAggregate([]error{err, patchStatusErr})
			err = fmt.Errorf("unable to patch status after DaemonSet reconciliation failed: %w", err)
		}
		return err
	}
	apimeta.SetStatusCondition(&vino.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             vinov1.ReconciliationSucceededReason,
		Message:            "DaemonSet reconciled",
		Type:               vinov1.ConditionTypeDaemonSetReady,
		ObservedGeneration: vino.GetGeneration(),
	})
	if err := r.patchStatus(ctx, vino); err != nil {
		err = fmt.Errorf("unable to patch status after DaemonSet reconciliation succeeded: %w", err)
		return err
	}

	return nil
}

func (r *VinoReconciler) ensureDaemonSet(ctx context.Context, vino *vinov1.Vino) error {
	ds, err := r.daemonSet(ctx, vino)
	if err != nil {
		return err
	}

	r.decorateDaemonSet(ctx, ds, vino)

	existDS := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, existDS)
	switch {
	case apierror.IsNotFound(err):
		err = r.Create(ctx, ds)
	case err == nil:
		err = r.Patch(ctx, ds, client.MergeFrom(existDS))
	}
	if err != nil {
		return err
	}

	// TODO (kkalynovskyi) this function needs to add owner reference on the daemonset set and watch
	// controller should watch for changes in daemonset to reconcile if it breaks, and change status
	// of the vino object
	// controlleruti.SetControllerReference(vino, ds, r.scheme)
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	return r.waitDaemonSet(ctx, ds)
}

func (r *VinoReconciler) decorateDaemonSet(ctx context.Context, ds *appsv1.DaemonSet, vino *vinov1.Vino) {
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
			logr.FromContext(ctx).Info("volume mount with vino spec is not found",
				"vino instance", vino.Namespace+"/"+vino.Name,
				"container name", c.Name,
			)
			ds.Spec.Template.Spec.Containers[i].VolumeMounts = append(
				ds.Spec.Template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					MountPath: "/vino/spec",
					Name:      volume,
					ReadOnly:  true,
					SubPath:   ConfigMapKeyVinoSpec,
				})
		}
	}

	// this will help avoid colisions if we have two vino CRs in the same namespace
	ds.Spec.Selector.MatchLabels[vinov1.VinoLabelDSNameSelector] = vino.Name
	ds.Spec.Template.ObjectMeta.Labels[vinov1.VinoLabelDSNameSelector] = vino.Name

	ds.Spec.Selector.MatchLabels[vinov1.VinoLabelDSNamespaceSelector] = vino.Namespace
	ds.Spec.Template.ObjectMeta.Labels[vinov1.VinoLabelDSNamespaceSelector] = vino.Namespace
}

func (r *VinoReconciler) waitDaemonSet(ctx context.Context, ds *appsv1.DaemonSet) error {
	logger := logr.FromContext(ctx).WithValues(
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
					logger.Info("DaemonSet is in ready status")
					return nil
				}
				logger.Info("DaemonSet is not in ready status, rechecking in 2 seconds")
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *VinoReconciler) daemonSet(ctx context.Context, vino *vinov1.Vino) (*appsv1.DaemonSet, error) {
	dsTemplate := vino.Spec.DaemonSetOptions.Template
	logger := logr.FromContext(ctx).WithValues("DaemonSetTemplate", dsTemplate)
	cm := &corev1.ConfigMap{}

	if dsTemplate == (vinov1.NamespacedName{}) {
		logger.Info("using default configmap for daemonset template")
		dsTemplate.Name = DaemonSetTemplateDefaultName
		dsTemplate.Namespace = DaemonSetTemplateDefaultNamespace
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
	if err := r.Delete(ctx,
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: vino.Name, Namespace: vino.Namespace,
			},
		}); err != nil {
		return err
	}
	if err := r.Delete(ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: vino.Name, Namespace: vino.Namespace,
			},
		}); err != nil {
		return err
	}
	controllerutil.RemoveFinalizer(vino, vinov1.VinoFinalizer)
	return r.Update(ctx, vino)
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
