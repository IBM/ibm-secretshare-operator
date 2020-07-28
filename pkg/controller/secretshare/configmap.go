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

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// getCm gets the configmap required to be copied
func (r *ReconcileSecretShare) getCm(name, ns string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// deleteCm deletes the copied configmap
func (r *ReconcileSecretShare) deleteCm(cmName, ns string) error {
	copiedCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: ns,
		},
	}
	if err := r.client.Delete(context.TODO(), copiedCm); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// createUpdateCm gets the Configmap required to be copied
func (r *ReconcileSecretShare) createUpdateCm(cm *corev1.ConfigMap, owner ownerutil.Owner) error {
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, cm, func() error {
		if owner != nil {
			ownerutil.EnsureOwner(cm, owner)
		}
		return nil
	})
	return err
}
