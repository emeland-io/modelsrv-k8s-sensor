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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"gitlab.com/emeland/k8s-model/internal/model"
)

var _ = Describe("APIInstanceReconciler", func() {
	var (
		ctx = context.Background()
		nn  = types.NamespacedName{Name: "my-svc", Namespace: "default"}
	)

	It("should create an APIInstance from a Service", func() {
		uid := uuid.New()
		apiID := uuid.New()
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nn.Name,
				Namespace: nn.Namespace,
				UID:       types.UID(uid.String()),
				Annotations: map[string]string{
					AnnotationAPIID: apiID.String(),
				},
			},
		}

		fakeClient := newFakeClient(svc)
		m := model.NewModel()

		r := NewAPIInstanceReconciler(fakeClient, testScheme, m, &corev1.Service{}, "Service")
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		ai := m.GetApiInstanceByResourceName(nn.String())
		Expect(ai).NotTo(BeNil())
		Expect(ai.DisplayName).To(Equal("my-svc"))
		Expect(ai.InstanceId).To(Equal(uid))
		Expect(ai.ApiRef.ApiID).To(Equal(apiID))
	})

	It("should create an APIInstance from an Ingress", func() {
		uid := uuid.New()
		ingNN := types.NamespacedName{Name: "my-ingress", Namespace: "default"}
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ingNN.Name,
				Namespace: ingNN.Namespace,
				UID:       types.UID(uid.String()),
			},
		}

		fakeClient := newFakeClient(ing)
		m := model.NewModel()

		r := NewAPIInstanceReconciler(fakeClient, testScheme, m, &networkingv1.Ingress{}, "Ingress")
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: ingNN})
		Expect(err).NotTo(HaveOccurred())

		ai := m.GetApiInstanceByResourceName(ingNN.String())
		Expect(ai).NotTo(BeNil())
		Expect(ai.InstanceId).To(Equal(uid))
	})

	It("should delete an APIInstance when the resource is gone", func() {
		m := model.NewModel()

		Expect(m.AddApiInstance(&model.APIInstance{
			DisplayName: "my-svc",
			InstanceId:  uuid.New(),
		}, nn.String(), nil)).To(Succeed())

		fakeClient := newFakeClient()
		r := NewAPIInstanceReconciler(fakeClient, testScheme, m, &corev1.Service{}, "Service")
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		Expect(m.GetApiInstanceByResourceName(nn.String())).To(BeNil())
	})
})
