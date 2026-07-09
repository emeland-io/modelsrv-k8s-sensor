/*
Copyright 2025 Lutz Behnke <lutz.behnke@emeland.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	clientgocache "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

// FindingRuleWatcher watches FindingRule resources and keeps the RuleRepo in sync.
type FindingRuleWatcher struct {
	Repo *RuleRepo
}

// +kubebuilder:rbac:groups=structure.emeland.io,resources=findingrules,verbs=get;list;watch

// NewFindingRuleWatcher registers a FindingRule watcher on the manager.
func NewFindingRuleWatcher(mgr ctrl.Manager, repo *RuleRepo) error {
	if repo == nil {
		return fmt.Errorf("rule repo is required")
	}
	return (&FindingRuleWatcher{Repo: repo}).SetupWithManager(mgr)
}

// SetupWithManager registers informer event handlers on the manager's cache.
func (w *FindingRuleWatcher) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	inf, err := mgr.GetCache().GetInformer(ctx, &structurev1alpha1.FindingRule{})
	if err != nil {
		return err
	}

	_, err = inf.AddEventHandler(clientgocache.ResourceEventHandlerFuncs{
		AddFunc:    w.onAdd,
		UpdateFunc: w.onUpdate,
		DeleteFunc: w.onDelete,
	})
	return err
}

func (w *FindingRuleWatcher) onAdd(obj interface{}) {
	w.loadRule(obj)
}

func (w *FindingRuleWatcher) onUpdate(_, newObj interface{}) {
	w.loadRule(newObj)
}

func (w *FindingRuleWatcher) onDelete(obj interface{}) {
	fr, ok := w.asFindingRule(obj)
	if !ok {
		return
	}
	w.Repo.Remove(fr.Name)
}

func (w *FindingRuleWatcher) loadRule(obj interface{}) {
	fr, ok := w.asFindingRule(obj)
	if !ok {
		return
	}
	if err := w.Repo.Set(fr); err != nil {
		logr.Log.WithName("findingrule-watcher").Error(
			err,
			"skipping FindingRule: CEL compilation failed",
			"name", fr.Name,
		)
	}
}

func (w *FindingRuleWatcher) asFindingRule(obj interface{}) (*structurev1alpha1.FindingRule, bool) {
	if tombstone, ok := obj.(clientgocache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	fr, ok := obj.(*structurev1alpha1.FindingRule)
	return fr, ok
}
