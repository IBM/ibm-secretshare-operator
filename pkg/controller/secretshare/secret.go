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

// getSecret gets the secret required to be copied
func (r *ReconcileSecretShare) getSecret(name, ns string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// createSecret gets the secret required to be copied
func (r *ReconcileSecretShare) createSecret(secret *corev1.Secret) error {
	if err := r.client.Create(context.TODO(), secret); err != nil {
		return err
	}
	return nil
}

// updateSecret gets the secret required to be copied
func (r *ReconcileSecretShare) updateSecret(secret *corev1.Secret) error {
	if err := r.client.Update(context.TODO(), secret); err != nil {
		return err
	}
	return nil
}

// createSecret gets the secret required to be copied
func (r *ReconcileSecretShare) createUpdateSecret(secret *corev1.Secret) error {
	existingSecret, err := r.getSecret(secret.Name, secret.Namespace)
	if err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		if err := r.createSecret(secret); err != nil {
			return err
		}
	} else {
		existingSecret.Data = secret.Data
		existingSecret.Type = secret.Type
		existingSecret.StringData = secret.StringData
		if err := r.updateSecret(existingSecret); err != nil {
			return err
		}
	}
	return nil
}
