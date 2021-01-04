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

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

// checkSub gets the subscription
func (r *SecretShareReconciler) checkSub(name, namespace string) bool {
	sub := &olmv1alpha1.Subscription{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, sub); err != nil {
		klog.Error(err)
		return false
	}
	return true
}
