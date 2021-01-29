//
// Copyright 2021 IBM Corporation
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

// addLabelstoSecret adds the secretshare labels for watching
func (r *SecretShareReconciler) addLabelstoSecret(secret *corev1.Secret, ss *ibmcpcsibmcomv1.SecretShare) error {
	existingSecret, err := r.getSecret(secret.Name, secret.Namespace)
	if err != nil {
		return err
	}
	if existingSecret.Labels == nil {
		existingSecret.Labels = make(map[string]string)
	}

	if existingSecret.Labels["secretshareName"] == ss.Name && existingSecret.Labels["secretshareNamespace"] == ss.Namespace {
		return nil
	}

	existingSecret.Labels["secretshareName"] = ss.Name
	existingSecret.Labels["secretshareNamespace"] = ss.Namespace
	if err := r.Client.Update(context.TODO(), existingSecret); err != nil {
		return err
	}
	return nil
}

// createSecret gets the secret required to be copied
func (r *SecretShareReconciler) createUpdateSecret(secret *corev1.Secret) error {
	existingSecret, err := r.getSecret(secret.Name, secret.Namespace)
	if existingSecret != nil {
		if reflect.DeepEqual(existingSecret.Data, secret.Data) && reflect.DeepEqual(existingSecret.StringData, secret.StringData) && reflect.DeepEqual(existingSecret.Type, secret.Type) {
			return nil
		}
		if err := r.Client.Update(context.TODO(), secret); err != nil {
			return err
		}
	} else if errors.IsNotFound(err) {
		if err := r.Client.Create(context.TODO(), secret); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}
