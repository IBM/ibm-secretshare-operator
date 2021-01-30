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

package controllers

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ibmcpcsibmcomv1 "github.com/IBM/ibm-secretshare-operator/api/v1"
)

// getCm gets the configmap required to be copied
func (r *SecretShareReconciler) getCm(name, ns string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// deleteCm deletes the copied configmap
func (r *SecretShareReconciler) deleteCm(cmName, ns string) error {
	copiedCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: ns,
		},
	}
	if err := r.Client.Delete(context.TODO(), copiedCm); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// addLabelstoConfigmap adds the secretshare labels for watching
func (r *SecretShareReconciler) addLabelstoConfigmap(cm *corev1.ConfigMap, ss *ibmcpcsibmcomv1.SecretShare) error {
	existingCm, err := r.getCm(cm.Name, cm.Namespace)
	if err != nil {
		return err
	}
	if existingCm.Labels == nil {
		existingCm.Labels = make(map[string]string)
	}

	if existingCm.Labels["secretshareName"] == ss.Name && existingCm.Labels["secretshareNamespace"] == ss.Namespace {
		return nil
	}

	existingCm.Labels["secretshareName"] = ss.Name
	existingCm.Labels["secretshareNamespace"] = ss.Namespace
	if err := r.Client.Update(context.TODO(), existingCm); err != nil {
		return err
	}
	return nil
}

// createUpdateCm gets the Configmap required to be copied
func (r *SecretShareReconciler) createUpdateCm(cm *corev1.ConfigMap) error {
	existingCm, err := r.getCm(cm.Name, cm.Namespace)
	if existingCm != nil {
		if reflect.DeepEqual(existingCm.Data, cm.Data) && reflect.DeepEqual(existingCm.BinaryData, cm.BinaryData) {
			return nil
		}
		if err := r.Client.Update(context.TODO(), cm); err != nil {
			return err
		}
	} else if errors.IsNotFound(err) {
		if err := r.Client.Create(context.TODO(), cm); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}
