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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// getCm gets the configmap required to be copied
func (r *ReconcileSecretShare) getCm(name, ns string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// createCm gets the configmap required to be copied
func (r *ReconcileSecretShare) createCm(cm *corev1.ConfigMap) error {
	if err := r.client.Create(context.TODO(), cm); err != nil {
		return err
	}
	return nil
}

// updateCm gets the configmap required to be copied
func (r *ReconcileSecretShare) updateCm(cm *corev1.ConfigMap) error {
	if err := r.client.Update(context.TODO(), cm); err != nil {
		return err
	}
	return nil
}

// createUpdateCm gets the Configmap required to be copied
func (r *ReconcileSecretShare) createUpdateCm(cm *corev1.ConfigMap) error {
	existingCm, err := r.getCm(cm.Name, cm.Namespace)
	if err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		if err := r.createCm(cm); err != nil {
			return err
		}
	} else {
		existingCm.Data = cm.Data
		existingCm.BinaryData = cm.BinaryData
		if err := r.updateCm(existingCm); err != nil {
			return err
		}
	}
	return nil
}
