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
	"os"
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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"

	vinov1 "vino/pkg/api/v1"
	"vino/pkg/ipam"
	"vino/pkg/managers"
)

const (
	TemplateDefaultKey           = "template"
	DaemonSetTemplateDefaultName = "vino-daemonset-template"

	ContainerNameLibvirt = "libvirt"
	ConfigMapKeyVinoSpec = "vino-spec"
)

// VinoReconciler reconciles a Vino object
type VinoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Ipam   *ipam.Ipam
}

// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=airship.airshipit.org,resources=vinoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=airship.airshipit.org,resources=ippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

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

func (r *VinoReconciler) getDaemonSetName(vino *vinov1.Vino) string {
	return fmt.Sprintf("%s-%s", vino.Namespace, vino.Name)
}

func (r *VinoReconciler) patchStatus(ctx context.Context, vino *vinov1.Vino) error {
	key := client.ObjectKeyFromObject(vino)
	latest := &vinov1.Vino{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}
	return r.Client.Status().Patch(ctx, vino, client.MergeFrom(latest))
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
	scheduledTimeoutCtx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	logger := logr.FromContext(ctx)
	logger.Info("Waiting for daemonset to become scheduled")
	if err = r.waitDaemonSet(scheduledTimeoutCtx, dsScheduled, ds); err != nil {
		return err
	}

	bmhManager := &managers.BMHManager{
		Namespace: getRuntimeNamespace(),
		ViNO:      vino,
		Client:    r.Client,
		Ipam:      r.Ipam,
		Logger:    logger,
	}

	logger.Info("Requesting Virtual Machines from vino-builders")
	if err := bmhManager.ScheduleVMs(ctx); err != nil {
		return err
	}

	waitTimeoutCtx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	logger.Info("Waiting for daemonset to become ready")
	if err := r.waitDaemonSet(waitTimeoutCtx, dsReady, ds); err != nil {
		return err
	}

	logger.Info("Creating BaremetalHosts")
	return bmhManager.CreateBMHs(ctx)
}

func (r *VinoReconciler) decorateDaemonSet(ctx context.Context, ds *appsv1.DaemonSet, vino *vinov1.Vino) {
	ds.Spec.Template.Spec.NodeSelector = vino.Spec.NodeSelector.MatchLabels
	ds.Namespace = getRuntimeNamespace()
	ds.Name = r.getDaemonSetName(vino)

	// TODO develop logic to derive all required ENV variables from VINO CR, and pass them
	// to setENV function instead
	if vino.Spec.VMBridge != "" {
		setEnv(ctx, ds, vinov1.EnvVarVMInterfaceName, vino.Spec.VMBridge)
	}
	setEnv(ctx, ds, vinov1.EnvVarBasicAuthUsername, vino.Spec.BMCCredentials.Username)
	setEnv(ctx, ds, vinov1.EnvVarBasicAuthPassword, vino.Spec.BMCCredentials.Password)

	// this will help avoid colisions if we have two vino CRs in the same namespace
	ds.Spec.Selector.MatchLabels[vinov1.VinoLabelDSNameSelector] = vino.Name
	ds.Spec.Template.ObjectMeta.Labels[vinov1.VinoLabelDSNameSelector] = vino.Name

	ds.Spec.Selector.MatchLabels[vinov1.VinoLabelDSNamespaceSelector] = vino.Namespace
	ds.Spec.Template.ObjectMeta.Labels[vinov1.VinoLabelDSNamespaceSelector] = vino.Namespace
}

// setEnv iterates over all container specs and sets the variable varName to
// varValue. If varName already exists for a container, setEnv overrides it
func setEnv(ctx context.Context, ds *appsv1.DaemonSet, varName, varValue string) {
	for i := range ds.Spec.Template.Spec.Containers {
		setContainerEnv(ctx, &ds.Spec.Template.Spec.Containers[i], varName, varValue)
	}
}

// setContainerEnv sets the variable varName to varValue for the container. If
// varName already exists for the container, setContainerEnv overrides it
func setContainerEnv(ctx context.Context, container *corev1.Container, varName, varValue string) {
	for j, envVar := range container.Env {
		if envVar.Name == varName {
			logr.FromContext(ctx).Info("found pre-existing environment variable on daemonset template, overriding it",
				"container name", container.Name,
				"environment variable", envVar.Name,
				"old value", envVar.Value,
				"new value", varValue,
			)
			container.Env[j].Value = varValue
			return
		}
	}

	// If we've made it this far, the variable didn't exist.
	container.Env = append(
		container.Env, corev1.EnvVar{
			Name:  varName,
			Value: varValue,
		},
	)
}

func (r *VinoReconciler) waitDaemonSet(ctx context.Context, check dsWaitCondition, ds *appsv1.DaemonSet) error {
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
				logger.Info("received error while waiting for ds to reach desired condition, sleeping",
					"error", err.Error())
			} else {
				logger.Info("checking daemonset status", "status", getDS.Status)
				if check(getDS) {
					logger.Info("DaemonSet is in ready status")
					return nil
				}
				logger.Info("DaemonSet is not in ready status, rechecking in 2 seconds")
			}
			time.Sleep(10 * time.Second)
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
		dsTemplate.Namespace = getRuntimeNamespace()
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

	template, exist := cm.Data[TemplateDefaultKey]
	if !exist {
		logger.Info("malformed template provided data doesn't have key " + TemplateDefaultKey)
		return nil, fmt.Errorf("malformed template provided data doesn't have key " + TemplateDefaultKey)
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
		For(&vinov1.Vino{}, builder.WithPredicates(
			predicate.GenerationChangedPredicate{},
		)).
		Complete(r)
}

func (r *VinoReconciler) finalize(ctx context.Context, vino *vinov1.Vino) error {
	bmhManager := &managers.BMHManager{
		Namespace: getRuntimeNamespace(),
		ViNO:      vino,
		Client:    r.Client,
		Ipam:      r.Ipam,
		Logger:    logr.FromContext(ctx),
	}
	if err := bmhManager.UnScheduleVMs(ctx); err != nil {
		return err
	}

	// TODO aggregate errors instead
	if err := r.Delete(ctx,
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.getDaemonSetName(vino), Namespace: getRuntimeNamespace(),
			},
		}); err != nil && !apierror.IsNotFound(err) {
		return err
	}

	controllerutil.RemoveFinalizer(vino, vinov1.VinoFinalizer)
	return r.Update(ctx, vino)
}

func getRuntimeNamespace() string {
	return os.Getenv("RUNTIME_NAMESPACE")
}

func dsScheduled(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled != 0 && ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled
}

func dsReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled != 0 && ds.Status.DesiredNumberScheduled == ds.Status.NumberReady
}

type dsWaitCondition func(ds *appsv1.DaemonSet) bool
