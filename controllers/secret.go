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

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// getSecret gets the secret required to be copied
func (r *SecretShareReconciler) getSecret(name, ns string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// deleteSecret deletes the copied secrets
func (r *SecretShareReconciler) deleteSecret(secretName, ns string) error {
	copiedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
	}
	if err := r.Client.Delete(context.TODO(), copiedSecret); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// createSecret gets the secret required to be copied
func (r *SecretShareReconciler) createUpdateSecret(secret *corev1.Secret, owner ownerutil.Owner) error {
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, secret, func() error {
		if owner != nil {
			ownerutil.EnsureOwner(secret, owner)
		}
		return nil
	})
	return err
}
