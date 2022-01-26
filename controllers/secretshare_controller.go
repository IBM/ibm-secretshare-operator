//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controllers

import (
	"context"
	"strings"
	"time"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ibmcpcsibmcomv1 "github.com/IBM/ibm-secretshare-operator/api/v1"
)

// SecretShareReconciler reconciles a SecretShare object
type SecretShareReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ibmcpcs.ibm.com,resources=secretshares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ibmcpcs.ibm.com,resources=secretshares/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=secret;configmap,verbs=get;update;patch;create;delete;list;watch

// Reconcile reads that state of the cluster for a SecretShare object and makes changes based on the state read
func (r *SecretShareReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	klog.V(1).Info("Reconciling SecretShare")

	// Fetch the SecretShare instance
	instance := &ibmcpcsibmcomv1.SecretShare{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	instance.InitStatus()

	if r.copySecretConfigmap(instance) {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
}

// copySecretConfigmap copies secret and configmap to the target namespace
func (r *SecretShareReconciler) copySecretConfigmap(instance *ibmcpcsibmcomv1.SecretShare) bool {
	requeueFromSecret := r.copySecret(instance)
	requeueFromConfigmap := r.copyConfigmap(instance)
	if err := r.Client.Status().Update(context.TODO(), instance); err != nil {
		klog.Errorf("Failed to update the status of the secretshare instance %s/%s : %v", instance.Namespace, instance.Name, err)
	}
	return requeueFromSecret || requeueFromConfigmap
}

// copySecret copies secret to the target namespace
func (r *SecretShareReconciler) copySecret(instance *ibmcpcsibmcomv1.SecretShare) bool {
	klog.V(1).Info("Copy secrets to the target namespace")
	ns := instance.Namespace
	secretList := instance.Spec.Secretshares
	requeue := false
	for _, secretShare := range secretList {
		running := true
		secretName := secretShare.Secretname
		secret, err := r.getSecret(secretName, ns)
		if err != nil {
			if errors.IsNotFound(err) && instance.CheckSecretStatus(ns+"/"+secretName, ibmcpcsibmcomv1.Running) {
				if r.deleteCopiedSecret(secretName, secretShare) {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsibmcomv1.Failed)
					requeue = true
					continue
				}
				instance.RemoveSecretStatus(ns + "/" + secretName)
			} else if !errors.IsNotFound(err) {
				klog.Error(err)
				instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsibmcomv1.Failed)
				requeue = true
				continue
			} else {
				if r.checkSub(secretOperatorMapping[secretName], ns) {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsibmcomv1.NotFound)
					requeue = true
					continue
				} else {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsibmcomv1.NotEnabled)
					continue
				}
			}
		} else {
			// Add labels for SecretShare instance
			if err := r.addLabelstoSecret(secret, instance); err != nil {
				klog.Error(err)
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsibmcomv1.Failed)
				requeue = true
				continue
			}
			for _, ns := range secretShare.Sharewith {
				if err := r.ensureNs(ns.Namespace); err != nil {
					klog.Error(err)
					running = false
					continue
				}
				if err := r.copySecretToTargetNs(secret, ns.Namespace); err != nil {
					klog.Error(err)
					running = false
					continue
				}
			}
			if running {
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsibmcomv1.Running)
			} else {
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsibmcomv1.Failed)
			}
		}
		requeue = requeue || !running
	}
	return requeue
}

// copyConfigmap copies configmap to the target namespace
func (r *SecretShareReconciler) copyConfigmap(instance *ibmcpcsibmcomv1.SecretShare) bool {
	klog.V(1).Info("Copy configmaps to the target namespace")
	ns := instance.Namespace
	cmList := instance.Spec.Configmapshares
	requeue := false
	for _, cmShare := range cmList {
		running := true
		cmName := cmShare.Configmapname
		cm, err := r.getCm(cmName, ns)
		if err != nil {
			if errors.IsNotFound(err) && instance.CheckConfigmapStatus(ns+"/"+cmName, ibmcpcsibmcomv1.Running) {
				if r.deleteCopiedCm(cmName, cmShare) {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsibmcomv1.Failed)
					requeue = true
					continue
				}
				instance.RemoveConfigmapStatus(ns + "/" + cmName)
			} else if !errors.IsNotFound(err) {
				klog.Error(err)
				instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsibmcomv1.Failed)
				requeue = true
				continue
			} else {
				if r.checkSub(cmOperatorMapping[cmName], ns) {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsibmcomv1.NotFound)
					requeue = true
					continue
				} else {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsibmcomv1.NotEnabled)
					continue
				}
			}
		} else {
			// Add labels for SecretShare instance
			if err := r.addLabelstoConfigmap(cm, instance); err != nil {
				klog.Error(err)
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsibmcomv1.Failed)
				requeue = true
				continue
			}
			for _, ns := range cmShare.Sharewith {
				if err := r.ensureNs(ns.Namespace); err != nil {
					klog.Error(err)
					running = false
					continue
				}
				if err := r.copyConfigmapToTargetNs(cm, ns.Namespace); err != nil {
					klog.Error(err)
					running = false
					continue
				}
			}
			if running {
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsibmcomv1.Running)
			} else {
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsibmcomv1.Failed)
			}
		}
		requeue = requeue || !running
	}
	return requeue
}

func (r *SecretShareReconciler) copySecretToTargetNs(secret *corev1.Secret, targetNs string) error {
	secretlabel := make(map[string]string)
	// Copy from the original labels to the target labels
	klog.Infof("Copy secret %s to %s namespace", secret.Name, targetNs)
	for k, v := range secret.Labels {
		secretlabel[k] = v
	}
	secretlabel["ibmcpcs.ibm.com/managed-by"] = "secretshare"
	secretlabel["manage-by-secretshare"] = "true"
	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: targetNs,
			Labels:    secretlabel,
		},
		Type:       secret.Type,
		Data:       secret.Data,
		StringData: secret.StringData,
	}
	if err := r.createUpdateSecret(targetSecret); err != nil {
		return err
	}
	return nil
}

func (r *SecretShareReconciler) copyConfigmapToTargetNs(cm *corev1.ConfigMap, targetNs string) error {
	cmlabel := make(map[string]string)
	// Copy from the original labels to the target labels
	klog.Infof("Copy configmap %s to %s namespace", cm.Name, targetNs)
	for k, v := range cm.Labels {
		cmlabel[k] = v
	}
	cmlabel["ibmcpcs.ibm.com/managed-by"] = "secretshare"
	cmlabel["manage-by-secretshare"] = "true"
	targetCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.Name,
			Namespace: targetNs,
			Labels:    cm.Labels,
		},
		Data:       cm.Data,
		BinaryData: cm.BinaryData,
	}
	if err := r.createUpdateCm(targetCm); err != nil {
		return err
	}
	return nil
}

func (r *SecretShareReconciler) deleteCopiedSecret(secretName string, secretShare ibmcpcsibmcomv1.Secretshare) bool {
	requeue := false
	for _, ns := range secretShare.Sharewith {
		if err := r.deleteSecret(secretName, ns.Namespace); err != nil {
			klog.Error(err)
			requeue = true
			continue
		}
	}
	return requeue
}

func (r *SecretShareReconciler) deleteCopiedCm(cmName string, cmShare ibmcpcsibmcomv1.Configmapshare) bool {
	requeue := false
	for _, ns := range cmShare.Sharewith {
		if err := r.deleteCm(cmName, ns.Namespace); err != nil {
			klog.Error(err)
			requeue = true
			continue
		}
	}
	return requeue
}

func getCMSecretToSS() handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		secretshare := []reconcile.Request{}
		labels := object.Meta.GetLabels()
		for ssKey, ss := range labels {
			if ss == "secretsharekey" {
				ssKeyList := strings.Split(ssKey, "/")
				if len(ssKeyList) != 2 {
					continue
				}
				secretshare = append(secretshare, reconcile.Request{NamespacedName: types.NamespacedName{Name: ssKeyList[1], Namespace: ssKeyList[0]}})
			}
		}
		return secretshare
	}
}

func getSecretShareMapper() handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		secretshare := []reconcile.Request{}
		if object.Meta.GetNamespace() == "ibm-common-services" {
			secretshare = append(secretshare, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-services", Namespace: "ibm-common-services"}})
		}
		return secretshare
	}
}

// SetupWithManager ...
func (r *SecretShareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	subPredicates := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Meta.GetNamespace() != "ibm-common-services"
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Meta.GetNamespace() != "ibm-common-services"
		},
	}
	cmsecretPredicates := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Meta.GetNamespace() != "ibm-common-services" {
				return false
			}
			labels := e.Meta.GetLabels()
			for labelKey := range labels {
				if labelKey == "manage-by-secretshare" {
					return true
				}
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.MetaNew.GetNamespace() != "ibm-common-services" {
				return false
			}
			labels := e.MetaNew.GetLabels()
			for labelKey := range labels {
				if labelKey == "manage-by-secretshare" {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if e.Meta.GetNamespace() != "ibm-common-services" {
				return false
			}
			labels := e.Meta.GetLabels()
			for labelKey := range labels {
				if labelKey == "manage-by-secretshare" {
					return true
				}
			}
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ibmcpcsibmcomv1.SecretShare{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: getCMSecretToSS()},
			builder.WithPredicates(cmsecretPredicates),
		).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: getCMSecretToSS()},
			builder.WithPredicates(cmsecretPredicates),
		).
		Watches(
			&source.Kind{Type: &olmv1alpha1.Subscription{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: getSecretShareMapper()},
			builder.WithPredicates(subPredicates),
		).
		Complete(r)
}
