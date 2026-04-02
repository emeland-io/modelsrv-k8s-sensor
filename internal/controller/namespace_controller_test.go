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

	"gitlab.com/emeland/k8s-model/internal/model"
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

		fakeClient := newFakeClient(ksNS)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: m}
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
		Expect(err).NotTo(HaveOccurred())

		emCtx := m.GetContextByResourceName("kube-system")
		Expect(emCtx).NotTo(BeNil())
		Expect(emCtx.ContextId).To(Equal(ksUID))
		Expect(emCtx.DisplayName).To(Equal("kube-system"))
		Expect(emCtx.ParentId).To(Equal(uuid.Nil)) // root has no parent
		Expect(r.getClusterContextID()).To(Equal(ksUID))
	})

	It("should create a child context from a regular namespace after kube-system", func() {
		ctx := context.Background()
		ksUID := uuid.New()
		nsUID := uuid.New()
		ksNS := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  types.UID(ksUID.String()),
			},
		}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
				UID:  types.UID(nsUID.String()),
				Annotations: map[string]string{
					"team": "platform",
				},
			},
		}

		fakeClient := newFakeClient(ksNS, ns)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: m}

		// Reconcile kube-system first to set the root context.
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
		Expect(err).NotTo(HaveOccurred())

		// Now reconcile the regular namespace.
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-app"}})
		Expect(err).NotTo(HaveOccurred())

		emCtx := m.GetContextByResourceName("my-app")
		Expect(emCtx).NotTo(BeNil())
		Expect(emCtx.ContextId).To(Equal(nsUID))
		Expect(emCtx.ParentId).To(Equal(ksUID))
		Expect(emCtx.Annotations["team"]).To(Equal("platform"))
	})

	It("should re-queue a regular namespace when kube-system is not yet seen", func() {
		ctx := context.Background()
		nsUID := uuid.New()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
				UID:  types.UID(nsUID.String()),
			},
		}

		fakeClient := newFakeClient(ns)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: m}
		result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-app"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeNumerically(">", 0))

		// Not added to model yet.
		Expect(m.GetContextByResourceName("my-app")).To(BeNil())
	})

	It("should delete a context when namespace is removed", func() {
		ctx := context.Background()
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		Expect(m.AddContext(&model.Context{
			DisplayName: "old-ns",
			ContextId:   uuid.New(),
		}, "old-ns")).To(Succeed())

		fakeClient := newFakeClient() // namespace gone
		r := &NamespaceReconciler{Client: fakeClient, Scheme: testScheme, Model: m}
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "old-ns"}})
		Expect(err).NotTo(HaveOccurred())

		Expect(m.GetContextByResourceName("old-ns")).To(BeNil())
	})
})
