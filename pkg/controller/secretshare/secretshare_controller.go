//
// Copyright 2020 IBM Corporation
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

package secretshare

import (
	"context"
	"time"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ibmcpcsv1 "github.com/IBM/ibm-secretshare-operator/pkg/apis/ibmcpcs/v1"
)

// Add creates a new SecretShare Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSecretShare{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secretshare-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SecretShare
	err = c.Watch(&source.Kind{Type: &ibmcpcsv1.SecretShare{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Secrets and requeue the owner SecretShare
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		OwnerType: &ibmcpcsv1.SecretShare{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Configmaps and requeue the owner SecretShare
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		OwnerType: &ibmcpcsv1.SecretShare{},
	})
	if err != nil {
		return err
	}

	// Watch for OperandConfig spec changes and requeue the OperandRequest
	if err = c.Watch(&source.Kind{Type: &olmv1alpha1.Subscription{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: getSecretShareMapper(),
	}, predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}); err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileSecretShare implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSecretShare{}

// ReconcileSecretShare reconciles a SecretShare object
type ReconcileSecretShare struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SecretShare object and makes changes based on the state read
func (r *ReconcileSecretShare) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.V(1).Info("Reconciling SecretShare")

	// Fetch the SecretShare instance
	instance := &ibmcpcsv1.SecretShare{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.InitStatus() {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	if r.copySecretConfigmap(instance) {
		return reconcile.Result{RequeueAfter: time.Second * 30}, nil
	}

	return reconcile.Result{}, nil
}

// copySecretConfigmap copies secret and configmap to the target namespace
func (r *ReconcileSecretShare) copySecretConfigmap(instance *ibmcpcsv1.SecretShare) bool {
	requeueFromSecret := r.copySecret(instance)
	requeueFromConfigmap := r.copyConfigmap(instance)
	return requeueFromSecret || requeueFromConfigmap
}

// copySecret copies secret to the target namespace
func (r *ReconcileSecretShare) copySecret(instance *ibmcpcsv1.SecretShare) bool {
	klog.V(1).Info("Copy secrets to the target namespace")
	ns := instance.Namespace
	secretList := instance.Spec.Secretshares
	requeue := false
	for _, secretShare := range secretList {
		running := true
		secretName := secretShare.Secretname
		secret, err := r.getSecret(secretName, ns)
		if err != nil {
			if errors.IsNotFound(err) && instance.CheckSecretStatus(ns+"/"+secretName, ibmcpcsv1.Running) {
				if r.deleteCopiedSecret(secretName, secretShare) {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsv1.Failed)
					requeue = true
					continue
				}
				instance.RemoveSecretStatus(ns + "/" + secretName)
			} else if !errors.IsNotFound(err) {
				klog.Error(err)
				instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsv1.Failed)
				requeue = true
				continue
			} else {
				if r.checkSub(secretOperatorMapping[secretName], ns) {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsv1.NotFound)
					requeue = true
					continue
				} else {
					instance.UpdateSecretStatus(ns+"/"+secretName, ibmcpcsv1.NotEnabled)
					continue
				}
			}
		} else {
			// Set SecretShare instance as the owner
			if err := r.createUpdateSecret(secret, instance); err != nil {
				klog.Error(err)
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsv1.Failed)
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
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsv1.Running)
			} else {
				instance.UpdateSecretStatus(secret.Namespace+"/"+secret.Name, ibmcpcsv1.Failed)
			}
		}
		requeue = requeue || !running
	}
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		requeue = true
	}
	return requeue
}

// copyConfigmap copies configmap to the target namespace
func (r *ReconcileSecretShare) copyConfigmap(instance *ibmcpcsv1.SecretShare) bool {
	klog.V(1).Info("Copy configmaps to the target namespace")
	ns := instance.Namespace
	cmList := instance.Spec.Configmapshares
	requeue := false
	for _, cmShare := range cmList {
		running := true
		cmName := cmShare.Configmapname
		cm, err := r.getCm(cmName, ns)
		if err != nil {
			if errors.IsNotFound(err) && instance.CheckConfigmapStatus(ns+"/"+cmName, ibmcpcsv1.Running) {
				if r.deleteCopiedCm(cmName, cmShare) {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsv1.Failed)
					requeue = true
					continue
				}
				instance.RemoveConfigmapStatus(ns + "/" + cmName)
			} else if !errors.IsNotFound(err) {
				klog.Error(err)
				instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsv1.Failed)
				requeue = true
				continue
			} else {
				if r.checkSub(cmOperatorMapping[cmName], ns) {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsv1.NotFound)
					requeue = true
					continue
				} else {
					instance.UpdateConfigmapStatus(ns+"/"+cmName, ibmcpcsv1.NotEnabled)
					continue
				}
			}
		} else {
			// Set SecretShare instance as the owner
			if err := r.createUpdateCm(cm, instance); err != nil {
				klog.Error(err)
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsv1.Failed)
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
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsv1.Running)
			} else {
				instance.UpdateConfigmapStatus(cm.Namespace+"/"+cm.Name, ibmcpcsv1.Failed)
			}
		}
		requeue = requeue || !running
	}
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		requeue = true
	}
	return requeue
}

func (r *ReconcileSecretShare) copySecretToTargetNs(secret *corev1.Secret, targetNs string) error {
	secretlabel := make(map[string]string)
	// Copy from the original labels to the target labels
	klog.Infof("Copy secret %s to %s namespace", secret.Name, targetNs)
	for k, v := range secret.Labels {
		secretlabel[k] = v
	}
	secretlabel["ibmcpcs.ibm.com/managed-by"] = "secretshare"
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
	if err := r.createUpdateSecret(targetSecret, nil); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileSecretShare) copyConfigmapToTargetNs(cm *corev1.ConfigMap, targetNs string) error {
	cmlabel := make(map[string]string)
	// Copy from the original labels to the target labels
	klog.Infof("Copy configmap %s to %s namespace", cm.Name, targetNs)
	for k, v := range cm.Labels {
		cmlabel[k] = v
	}
	cmlabel["ibmcpcs.ibm.com/managed-by"] = "secretshare"
	targetCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.Name,
			Namespace: targetNs,
			Labels:    cm.Labels,
		},
		Data:       cm.Data,
		BinaryData: cm.BinaryData,
	}
	if err := r.createUpdateCm(targetCm, nil); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileSecretShare) deleteCopiedSecret(secretName string, secretShare ibmcpcsv1.Secretshare) bool {
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

func (r *ReconcileSecretShare) deleteCopiedCm(cmName string, cmShare ibmcpcsv1.Configmapshare) bool {
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

func getSecretShareMapper() handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		secretshare := []reconcile.Request{}
		secretshare = append(secretshare, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-services", Namespace: "ibm-common-services"}})
		return secretshare
	}
}
