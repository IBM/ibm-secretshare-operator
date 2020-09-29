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

package utils

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	cr_cache "sigs.k8s.io/controller-runtime/pkg/cache"
	cr_client "sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrUnsupported is returned for unsupported operations
var ErrUnsupported = errors.New("unsupported operation")

// ErrInternalError is returned for unexpected errors
var ErrInternalError = errors.New("internal error")

var secretSchema = corev1.SchemeGroupVersion.WithKind("Secret")
var secretListSchema = corev1.SchemeGroupVersion.WithKind("SecretList")
var cmSchema = corev1.SchemeGroupVersion.WithKind("ConfigMap")
var cmListSchema = corev1.SchemeGroupVersion.WithKind("ConfigMapList")

// New constructs a new Cache with custom logic to handle secrets
func New(config *rest.Config, opts cr_cache.Options, optionsModifier func(options *metav1.ListOptions)) (cr_cache.Cache, error) {
	// Setup filtered secret informer that will only store/return items matching the filter for listing purposes
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err, "Failed to construct client")
		return nil, err
	}
	var resync time.Duration
	if opts.Resync != nil {
		resync = *opts.Resync
	}
	secretlistFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		optionsModifier(&options)
		result, err := clientSet.CoreV1().Secrets(opts.Namespace).List(context.TODO(), options)
		if err != nil {
			klog.Info("Failed to list secrets", "error", err)
		}
		return result, err
	}
	secretwatchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		optionsModifier(&options)
		return clientSet.CoreV1().Secrets(opts.Namespace).Watch(context.TODO(), options)
	}

	secretlisterWatcher := &cache.ListWatch{ListFunc: secretlistFunc, WatchFunc: secretwatchFunc}
	secretInformer := cache.NewSharedIndexInformer(secretlisterWatcher, &corev1.Secret{}, resync, cache.Indexers{})

	cmlistFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		optionsModifier(&options)
		result, err := clientSet.CoreV1().ConfigMaps(opts.Namespace).List(context.TODO(), options)
		if err != nil {
			klog.Info("Failed to list configmaps", "error", err)
		}
		return result, err
	}
	cmwatchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		optionsModifier(&options)
		return clientSet.CoreV1().ConfigMaps(opts.Namespace).Watch(context.TODO(), options)
	}
	cmlisterWatcher := &cache.ListWatch{ListFunc: cmlistFunc, WatchFunc: cmwatchFunc}
	cmInformer := cache.NewSharedIndexInformer(cmlisterWatcher, &corev1.ConfigMap{}, resync, cache.Indexers{})
	// Setup regular cache to handle other resource kinds
	fallback, err := cr_cache.New(config, opts)
	if err != nil {
		klog.Error(err, "Failed to init fallback cache")
		return nil, err
	}
	return customCache{clientSet: clientSet, secretInformer: secretInformer, cmInformer: cmInformer, fallback: fallback}, nil
}

type customCache struct {
	clientSet      *kubernetes.Clientset
	secretInformer cache.SharedIndexInformer
	cmInformer     cache.SharedIndexInformer
	fallback       cr_cache.Cache
}

func (cc customCache) Get(ctx context.Context, key cr_client.ObjectKey, obj runtime.Object) error {
	if secret, ok := obj.(*corev1.Secret); ok {
		// Check store, then real client
		if err := cc.getSecretFromStore(key, secret); err == nil {
			// Do nothing
		} else if err := cc.getSecretFromClient(ctx, key, secret); err != nil {
			return err
		}
		secret.SetGroupVersionKind(secretSchema)
		return nil
	}
	if _, ok := obj.(*corev1.SecretList); ok {
		klog.Info("Get with SecretList object unsupported")
		return ErrUnsupported
	}
	if cm, ok := obj.(*corev1.ConfigMap); ok {
		// Check store, then real client
		if err := cc.getConfigMapFromStore(key, cm); err == nil {
			// Do nothing
		} else if err := cc.getConfigMapFromClient(ctx, key, cm); err != nil {
			return err
		}
		cm.SetGroupVersionKind(cmSchema)
		return nil
	}
	if _, ok := obj.(*corev1.ConfigMapList); ok {
		klog.Info("Get with ConfigMapList object unsupported")
		return ErrUnsupported
	}
	// Passthrough
	return cc.fallback.Get(ctx, key, obj)
}

func (cc customCache) getSecretFromStore(key cr_client.ObjectKey, obj *corev1.Secret) error {
	item, exists, err := cc.secretInformer.GetStore().GetByKey(key.String())
	if err != nil {
		klog.Info("Failed to get item from cache", "error", err)
		return ErrInternalError
	}
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: secretSchema.Group, Resource: "secrets"}, key.String())
	}
	result, ok := item.(*corev1.Secret)
	if !ok {
		klog.Info("Failed to convert secret", "item", result)
		return ErrInternalError
	}

	result.DeepCopyInto(obj)
	return nil
}

func (cc customCache) getConfigMapFromStore(key cr_client.ObjectKey, obj *corev1.ConfigMap) error {
	item, exists, err := cc.cmInformer.GetStore().GetByKey(key.String())
	if err != nil {
		klog.Info("Failed to get item from cache", "error", err)
		return ErrInternalError
	}
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: cmSchema.Group, Resource: "configmaps"}, key.String())
	}
	result, ok := item.(*corev1.ConfigMap)
	if !ok {
		klog.Info("Failed to convert configmap", "item", result)
		return ErrInternalError
	}
	result.DeepCopyInto(obj)
	return nil
}

func (cc customCache) getSecretFromClient(ctx context.Context, key cr_client.ObjectKey, obj *corev1.Secret) error {

	result, err := cc.clientSet.CoreV1().Secrets(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		klog.Info("Failed to retrieve secret", "error", err)
		return err
	}

	result.DeepCopyInto(obj)
	return nil
}

func (cc customCache) getConfigMapFromClient(ctx context.Context, key cr_client.ObjectKey, obj *corev1.ConfigMap) error {

	result, err := cc.clientSet.CoreV1().ConfigMaps(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		klog.Info("Failed to retrieve configmap", "error", err)
		return err
	}
	result.DeepCopyInto(obj)
	return nil
}

func (cc customCache) List(ctx context.Context, list runtime.Object, opts ...cr_client.ListOption) error {
	if _, ok := list.(*corev1.Secret); ok {
		klog.Info("List with Secret object unsupported")
		return ErrUnsupported
	}
	if secretList, ok := list.(*corev1.SecretList); ok {
		// Construct filter
		listOpts := cr_client.ListOptions{}
		for _, opt := range opts {
			opt.ApplyToList(&listOpts)
		}
		if listOpts.LabelSelector == nil || listOpts.LabelSelector.Empty() {
			klog.Info("Warning! Unfiltered List call. List only returns items watched by the filtered informer")
		}
		// Construct result
		result := cc.secretInformer.GetStore().List()
		secretList.Items = make([]corev1.Secret, 0, len(result))
		// Filter items
		for _, item := range result {
			if secret, ok := item.(*corev1.Secret); ok && cc.secretMatchesOptions(secret, listOpts) {
				copy := secret.DeepCopy()
				copy.SetGroupVersionKind(secretSchema)
				secretList.Items = append(secretList.Items, *copy)
			}
		}
		secretList.SetGroupVersionKind(secretListSchema)
		klog.Info("Secret list filtered", "namespace", listOpts.Namespace, "filter", listOpts.LabelSelector, "all", len(result), "filtered", len(secretList.Items))
		return nil
	}
	if _, ok := list.(*corev1.Secret); ok {
		klog.Info("List with Secret object unsupported")
		return ErrUnsupported
	}

	if cmList, ok := list.(*corev1.ConfigMapList); ok {
		// Construct filter
		listOpts := cr_client.ListOptions{}
		for _, opt := range opts {
			opt.ApplyToList(&listOpts)
		}
		if listOpts.LabelSelector == nil || listOpts.LabelSelector.Empty() {
			klog.Info("Warning! Unfiltered List call. List only returns items watched by the filtered informer")
		}
		// Construct result
		result := cc.cmInformer.GetStore().List()
		cmList.Items = make([]corev1.ConfigMap, 0, len(result))
		// Filter items
		for _, item := range result {
			if cm, ok := item.(*corev1.ConfigMap); ok && cc.configMapMatchesOptions(cm, listOpts) {
				copy := cm.DeepCopy()
				copy.SetGroupVersionKind(cmSchema)
				cmList.Items = append(cmList.Items, *copy)
			}
		}
		cmList.SetGroupVersionKind(cmListSchema)
		klog.Info("ConfigMap list filtered", "namespace", listOpts.Namespace, "filter", listOpts.LabelSelector, "all", len(result), "filtered", len(cmList.Items))
		return nil
	}
	// Passthrough
	return cc.fallback.List(ctx, list, opts...)
}

func (cc customCache) secretMatchesOptions(secret *corev1.Secret, opt cr_client.ListOptions) bool {
	if opt.Namespace != "" && secret.Namespace != opt.Namespace {
		return false
	}
	if opt.FieldSelector != nil && !opt.FieldSelector.Empty() {
		klog.Info("Field selector for SecretList not supported")
	}
	if opt.LabelSelector != nil && !opt.LabelSelector.Empty() {
		if !opt.LabelSelector.Matches(labels.Set(secret.Labels)) {
			return false
		}
	}
	return true
}

func (cc customCache) configMapMatchesOptions(cm *corev1.ConfigMap, opt cr_client.ListOptions) bool {
	if opt.Namespace != "" && cm.Namespace != opt.Namespace {
		return false
	}
	if opt.FieldSelector != nil && !opt.FieldSelector.Empty() {
		klog.Info("Field selector for ConfigMapList not supported")
	}
	if opt.LabelSelector != nil && !opt.LabelSelector.Empty() {
		if !opt.LabelSelector.Matches(labels.Set(cm.Labels)) {
			return false
		}
	}
	return true
}

func (cc customCache) GetInformer(ctx context.Context, obj runtime.Object) (cr_cache.Informer, error) {
	if _, ok := obj.(*corev1.Secret); ok {
		return cc.secretInformer, nil
	}
	if _, ok := obj.(*corev1.SecretList); ok {
		return cc.secretInformer, nil
	}
	if _, ok := obj.(*corev1.ConfigMap); ok {
		return cc.cmInformer, nil
	}
	if _, ok := obj.(*corev1.ConfigMapList); ok {
		return cc.cmInformer, nil
	}
	// Passthrough
	return cc.fallback.GetInformer(ctx, obj)
}

func (cc customCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cr_cache.Informer, error) {
	if gvk == corev1.SchemeGroupVersion.WithKind("Secret") {
		return cc.secretInformer, nil
	}
	if gvk == corev1.SchemeGroupVersion.WithKind("SecretList") {
		return cc.secretInformer, nil
	}
	if gvk == corev1.SchemeGroupVersion.WithKind("ConfigMap") {
		return cc.cmInformer, nil
	}
	if gvk == corev1.SchemeGroupVersion.WithKind("ConfigMapList") {
		return cc.cmInformer, nil
	}
	// Passthrough
	return cc.fallback.GetInformerForKind(ctx, gvk)
}

func (cc customCache) Start(stopCh <-chan struct{}) error {
	klog.Info("Start")
	go cc.secretInformer.Run(stopCh)
	go cc.cmInformer.Run(stopCh)
	return cc.fallback.Start(stopCh)
}

func (cc customCache) WaitForCacheSync(stop <-chan struct{}) bool {
	// Wait for secret informer to sync
	klog.Info("Waiting for secret and configmap informer to sync")
	waiting := true
	for waiting {
		select {
		case <-stop:
			waiting = false
		case <-time.After(time.Second):
			waiting = !cc.cmInformer.HasSynced() && !cc.secretInformer.HasSynced()
		}
	}
	// Wait for fallback cache to sync
	klog.Info("Waiting for fallback informer to sync")
	return cc.fallback.WaitForCacheSync(stop)
}

func (cc customCache) IndexField(ctx context.Context, obj runtime.Object, field string, extractValue cr_client.IndexerFunc) error {
	if _, ok := obj.(*corev1.Secret); ok {
		klog.Info("IndexField for Secret not supported")
		return ErrUnsupported
	}
	if _, ok := obj.(*corev1.SecretList); ok {
		klog.Info("IndexField for SecretList not supported")
		return ErrUnsupported
	}
	if _, ok := obj.(*corev1.ConfigMap); ok {
		klog.Info("IndexField for ConfigMap not supported")
		return ErrUnsupported
	}
	if _, ok := obj.(*corev1.ConfigMapList); ok {
		klog.Info("IndexField for ConfigMapList not supported")
		return ErrUnsupported
	}
	return cc.fallback.IndexField(ctx, obj, field, extractValue)
}
