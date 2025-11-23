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
	"gitlab.com/emeland/k8s-model/internal/model"
)

var _ = Describe("Component Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const displayName = "Test Component"
		componentId := uuid.New()
		const availableDate = "2023-01-01"
		const deprecatedDate = "2024-01-01"
		const terminatedDate = "2025-01-01"
		const description = "This is a test Component."
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
		component := &v1alpha1.Component{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Component")
			err := k8sClient.Get(ctx, typeNamespacedName, component)
			if err != nil && errors.IsNotFound(err) {
				resource := &v1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Annotations: map[string]string{
							"structure.emeland.io/system-id": "test-system-id",
						},
					},
					Spec: v1alpha1.ComponentSpec{
						DisplayName: displayName,
						Description: description,
						ComponentId: componentId.String(),
						Version:     version,
						Consumes: []v1alpha1.APIRef{
							{
								ApiId: "test-consumed-api",
							},
						},
						Provides: []v1alpha1.APIRef{
							{
								ApiId: "test-provided-api",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &v1alpha1.Component{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Component")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			model, err := model.NewModel()
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &ComponentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Model:  model,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			compInfo := model.GetComponentByResourceName("no-such-resource")
			Expect(compInfo).To(BeNil())

			compInfo = model.GetComponentByResourceName(typeNamespacedName.String())
			Expect(compInfo).NotTo(BeNil())

			comp := compInfo.Component
			Expect(comp).NotTo(BeNil())
			Expect(comp.DisplayName).To(Equal(displayName))
			Expect(comp.Description).To(Equal(description))
			Expect(comp.ComponentId).To(Equal(componentId))
			Expect(comp.Version).To(Equal(parseVersion(version)))
			// TODO: add test with system set
			Expect(comp.System).To(BeNil())
			Expect(comp.Annotations["structure.emeland.io/system-id"]).To(Equal("test-system-id"))

		})
	})
})
