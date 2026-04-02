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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"gitlab.com/emeland/k8s-model/internal/model"
)

var _ = Describe("WorkloadReconciler", func() {
	var (
		ctx = context.Background()
		nn  = types.NamespacedName{Name: "my-deploy", Namespace: "default"}
	)

	It("should create a ComponentInstance from a Deployment", func() {
		uid := uuid.New()
		siID := uuid.New()
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nn.Name,
				Namespace: nn.Namespace,
				UID:       types.UID(uid.String()),
				Annotations: map[string]string{
					AnnotationSystemInstanceID: siID.String(),
					"app":                      "test",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			},
		}

		fakeClient := newFakeClient(dep)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		r := NewWorkloadReconciler(fakeClient, testScheme, m, &appsv1.Deployment{}, "Deployment", nil)
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		ci := m.GetComponentInstanceByResourceName(nn.String())
		Expect(ci).NotTo(BeNil())
		Expect(ci.DisplayName).To(Equal("my-deploy"))
		Expect(ci.InstanceId).To(Equal(uid))
		Expect(ci.SystemInstance.InstanceId).To(Equal(siID))
		Expect(ci.Annotations["app"]).To(Equal("test"))
	})

	It("should delete a ComponentInstance when the resource is gone", func() {
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		// Pre-populate model
		Expect(m.AddComponentInstance(&model.ComponentInstance{
			DisplayName: "my-deploy",
			InstanceId:  uuid.New(),
		}, nn.String(), nil)).To(Succeed())
		Expect(m.GetComponentInstanceByResourceName(nn.String())).NotTo(BeNil())

		fakeClient := newFakeClient() // resource doesn't exist
		r := NewWorkloadReconciler(fakeClient, testScheme, m, &appsv1.Deployment{}, "Deployment", nil)
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		Expect(m.GetComponentInstanceByResourceName(nn.String())).To(BeNil())
	})

	It("should skip Jobs owned by a CronJob", func() {
		uid := uuid.New()
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cron-job-12345",
				Namespace: "default",
				UID:       types.UID(uid.String()),
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "CronJob", Name: "my-cron", APIVersion: "batch/v1"},
				},
			},
		}

		fakeClient := newFakeClient(job)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		jobNN := types.NamespacedName{Name: "cron-job-12345", Namespace: "default"}
		r := NewWorkloadReconciler(fakeClient, testScheme, m, &batchv1.Job{}, "Job", IsOwnedByCronJob)
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: jobNN})
		Expect(err).NotTo(HaveOccurred())

		Expect(m.GetComponentInstanceByResourceName(jobNN.String())).To(BeNil())
	})

	It("should track standalone Jobs", func() {
		uid := uuid.New()
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standalone-job",
				Namespace: "default",
				UID:       types.UID(uid.String()),
			},
		}

		fakeClient := newFakeClient(job)
		m, err := model.NewModel()
		Expect(err).NotTo(HaveOccurred())

		jobNN := types.NamespacedName{Name: "standalone-job", Namespace: "default"}
		r := NewWorkloadReconciler(fakeClient, testScheme, m, &batchv1.Job{}, "Job", IsOwnedByCronJob)
		_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: jobNN})
		Expect(err).NotTo(HaveOccurred())

		ci := m.GetComponentInstanceByResourceName(jobNN.String())
		Expect(ci).NotTo(BeNil())
		Expect(ci.InstanceId).To(Equal(uid))
	})
})
