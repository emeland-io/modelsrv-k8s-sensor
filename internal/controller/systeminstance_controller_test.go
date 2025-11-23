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
	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/model"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SystemInstance Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-system-instance"
		const systemName = "test-system"
		const displayName = "Test System Instance"
		instanceId := uuid.New()
		systemId := uuid.New()

		ctx := context.Background()

		systemNsName := types.NamespacedName{
			Name:      systemName,
			Namespace: "default",
		}
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		system := &v1alpha1.System{}
		systeminstance := &v1alpha1.SystemInstance{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind SystemInstance")
			err := k8sClient.Get(ctx, systemNsName, system)
			if err != nil && errors.IsNotFound(err) {
				resource := &v1alpha1.System{
					ObjectMeta: metav1.ObjectMeta{
						Name:      systemName,
						Namespace: "default",
						Annotations: map[string]string{
							"structure.emeland.io/system-id": systemId.String(),
						}},
					Spec: v1alpha1.SystemSpec{
						SystemId:    systemId.String(),
						DisplayName: "Test System",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			err = k8sClient.Get(ctx, typeNamespacedName, systeminstance)
			if err != nil && errors.IsNotFound(err) {
				resource := &v1alpha1.SystemInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Annotations: map[string]string{
							"structure.emeland.io/system-id": "test-system-id",
						},
					},
					Spec: v1alpha1.SystemInstanceSpec{
						InstanceId:  instanceId.String(),
						SystemId:    systemId.String(),
						DisplayName: displayName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &v1alpha1.SystemInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SystemInstance")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			model, err := model.NewModel()
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &SystemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Model:  model,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			sysInstance := model.GetSystemInstanceByResourceName("no-such-resource")
			Expect(sysInstance).To(BeNil())

			sysInstance = model.GetSystemInstanceByResourceName(typeNamespacedName.String())
			Expect(sysInstance).NotTo(BeNil())
			Expect(sysInstance.DisplayName).To(Equal(displayName))
			Expect(sysInstance.InstanceId).To(Equal(instanceId))
			Expect(sysInstance.SystemRef).ToNot(BeNil())
			Expect(sysInstance.SystemRef.SystemId).To(Equal(systemId))
			Expect(sysInstance.Annotations["structure.emeland.io/system-id"]).To(Equal("test-system-id"))
		})
	})
})
