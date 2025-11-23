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
	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/model"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

var _ = Describe("API Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const displayName = "Test API"
		apiId := uuid.New()
		apiType := modelsrv.ParseApiType("OpenAPI")
		const availableDate = "2023-01-01"
		const deprecatedDate = "2024-01-01"
		const terminatedDate = "2025-01-01"
		const description = "This is a test API."
		version := v1alpha1.Version{
			Version:        "1.0.0",
			AvailableFrom:  availableDate,
			DeprecatedFrom: deprecatedDate,
			TerminatedFrom: terminatedDate,
		}

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		api := &structurev1alpha1.API{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind API")
			err := k8sClient.Get(ctx, typeNamespacedName, api)
			if err != nil && errors.IsNotFound(err) {
				resource := &structurev1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Annotations: map[string]string{
							"structure.emeland.io/system-id": "test-system-id",
						},
					},
					Spec: structurev1alpha1.APISpec{
						DisplayName: displayName,
						Description: description,
						ApiId:       apiId.String(),
						Version:     version,
						Type:        apiType.String(),
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &structurev1alpha1.API{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance API")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			model, err := model.NewModel()
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &APIReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Model:  model,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			apiInfo := model.GetApiByResourceName("no-such-resource")
			Expect(apiInfo).To(BeNil())

			apiInfo = model.GetApiByResourceName(typeNamespacedName.String())
			Expect(apiInfo).NotTo(BeNil())

			api := apiInfo.API
			Expect(api.DisplayName).To(Equal(displayName))
			Expect(api.Description).To(Equal(description))
			Expect(api.ApiId).To(Equal(apiId))
			Expect(api.Version).To(Equal(parseVersion(version)))
			Expect(api.Type).To(Equal(apiType))
			// TODO: add test with system set
			Expect(api.System).To(BeNil())
			Expect(api.Annotations["structure.emeland.io/system-id"]).To(Equal("test-system-id"))

		})
	})
})
