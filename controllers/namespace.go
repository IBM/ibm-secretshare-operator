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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ensureNs makes sure if the target namespace exist
func (r *SecretShareReconciler) ensureNs(ns string) error {
	if err := r.getNs(ns); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := r.createNs(ns); err != nil {
			return err
		}
	}
	return nil
}

// getNs gets the target namespace
func (r *SecretShareReconciler) getNs(ns string) error {
	targetNs := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: ns}, targetNs); err != nil {
		return err
	}
	return nil
}

// createNs creates the target namespace
func (r *SecretShareReconciler) createNs(ns string) error {
	targetNs := &corev1.Namespace{}
	targetNs.SetName(ns)
	if err := r.Client.Create(context.TODO(), targetNs); err != nil {
		return err
	}
	return nil
}
