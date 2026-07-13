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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.emeland.io/modelsrv/pkg/backend"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
)

var _ = Describe("NamespaceReconciler", func() {
	It("should create a root context from kube-system", func() {
		ctx := context.Background()
		ksUID := uuid.New()
		ksNS := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  types.UID(ksUID.String()),
			},
		}

		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()
		fakeClient := newFakeClient(ksNS)

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: b.GetModel(), Index: idx}
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
		Expect(err).NotTo(HaveOccurred())

		emCtx := b.GetModel().GetContextById(ksUID)
		Expect(emCtx).NotTo(BeNil())
		Expect(emCtx.GetContextId()).To(Equal(ksUID))
		Expect(emCtx.GetDisplayName()).To(Equal("kube-system"))
		Expect(emCtx.GetParentId()).To(Equal(uuid.Nil))
		Expect(r.getClusterContextID()).To(Equal(ksUID))
	})

	It("should create a child context from a regular namespace after kube-system", func() {
		ctx := context.Background()
		ksUID := uuid.New()
		nsUID := uuid.New()
		ksNS := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID(ksUID.String())},
		}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "my-app",
				UID:         types.UID(nsUID.String()),
				Annotations: map[string]string{"team": "platform"},
			},
		}

		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()
		fakeClient := newFakeClient(ksNS, ns)

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: b.GetModel(), Index: idx}

		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
		Expect(err).NotTo(HaveOccurred())

		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-app"}})
		Expect(err).NotTo(HaveOccurred())

		emCtx := b.GetModel().GetContextById(nsUID)
		Expect(emCtx).NotTo(BeNil())
		Expect(emCtx.GetContextId()).To(Equal(nsUID))
		Expect(emCtx.GetParentId()).To(Equal(ksUID))
		Expect(emCtx.GetAnnotations().GetValue("team")).To(Equal("platform"))
	})

	It("should re-queue a regular namespace when kube-system is not yet seen", func() {
		ctx := context.Background()
		nsUID := uuid.New()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "my-app", UID: types.UID(nsUID.String())},
		}

		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		fakeClient := newFakeClient(ns)

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: b.GetModel(), Index: NewNameIndex()}
		result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-app"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeNumerically(">", 0))

		Expect(b.GetModel().GetContextById(nsUID)).To(BeNil())
	})

	It("should delete a context when namespace is removed", func() {
		ctx := context.Background()
		id := uuid.New()
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()

		emCtx := mdlctx.NewContext(id)
		emCtx.SetDisplayName("old-ns")
		Expect(b.GetModel().AddContext(emCtx)).To(Succeed())
		idx.Put(KindContext, "old-ns", id)

		fakeClient := newFakeClient()
		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: b.GetModel(), Index: idx}
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "old-ns"}})
		Expect(err).NotTo(HaveOccurred())

		Expect(b.GetModel().GetContextById(id)).To(BeNil())
	})

	It("should evaluate FindingRules and create findings on reconcile", func() {
		ctx := context.Background()
		ksUID := uuid.New()
		nsUID := uuid.New()
		ksNS := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID(ksUID.String())},
		}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
				UID:  types.UID(nsUID.String()),
			},
		}

		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()
		fakeClient := newFakeClient(ksNS, ns)

		repo := NewRuleRepo()
		Expect(repo.Set(namespaceFindingRule("ns-rule", "true"))).To(Succeed())

		r := &NamespaceReconciler{
			Client:   fakeClient,
			Scheme:   testScheme,
			Model:    b.GetModel(),
			Index:    idx,
			RuleEval: NewRuleEvaluation(repo, NewEvaluator(b.GetModel()), "/namespaces"),
		}

		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
		Expect(err).NotTo(HaveOccurred())

		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-app"}})
		Expect(err).NotTo(HaveOccurred())

		findings, err := b.GetModel().GetFindings()
		Expect(err).NotTo(HaveOccurred())

		expectedIDs := map[uuid.UUID]struct{}{
			evaluatorFindingID("ns-rule", types.UID(ksUID.String())): {},
			evaluatorFindingID("ns-rule", types.UID(nsUID.String())): {},
		}
		matched := 0
		for _, f := range findings {
			if _, ok := expectedIDs[f.GetFindingId()]; ok {
				matched++
				Expect(f.GetDisplayName()).To(Equal("TestFinding"))
			}
		}
		Expect(matched).To(Equal(2))
	})
})
