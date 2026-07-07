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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"go.emeland.io/modelsrv/pkg/backend"
)

var _ = Describe("System Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const displayName = "Test System"
		systemId := uuid.New()
		const availableDate = "2023-01-01"
		const deprecatedDate = "2024-01-01"
		const terminatedDate = "2025-01-01"
		const description = "This is a test system."
		version := v1alpha1.Version{
			Version:        "1.0.0",
			AvailableFrom:  availableDate,
			DeprecatedFrom: deprecatedDate,
			TerminatedFrom: terminatedDate,
		}

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		system := &v1alpha1.System{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind System")
			err := k8sClient.Get(ctx, typeNamespacedName, system)
			if err != nil && errors.IsNotFound(err) {
				resource := &v1alpha1.System{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Annotations: map[string]string{
							"structure.emeland.io/system-id": "test-system-id",
						},
					},
					Spec: v1alpha1.SystemSpec{
						DisplayName: displayName,
						Description: description,
						SystemId:    systemId.String(),
						Version:     version,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &v1alpha1.System{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)
			By("Cleanup the specific resource instance System")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			controllerReconciler := &SystemReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Model:  b.GetModel(),
				Index:  idx,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			sys := b.GetModel().GetSystemById(systemId)
			Expect(sys).NotTo(BeNil())
			Expect(sys.GetDisplayName()).To(Equal(displayName))
			Expect(sys.GetDescription()).To(Equal(description))
			Expect(sys.GetSystemId()).To(Equal(systemId))
			Expect(sys.GetAnnotations().GetValue("structure.emeland.io/system-id")).To(Equal("test-system-id"))
		})
	})
})
