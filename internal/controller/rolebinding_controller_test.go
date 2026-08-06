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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

var _ = Describe("RoleBinding Controller", func() {
	Context("When reconciling a RoleBinding", func() {
		const bindingName = "test-rolebinding"
		const bindingNamespace = "default"
		subjectID := uuid.MustParse("11111111-2222-4333-8444-555555555555")

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: bindingName, Namespace: bindingNamespace}

		It("should map a RoleBinding with subject annotation to an EmELand Binding", func() {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: bindingNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: subjectID.String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "some-role",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "developers", APIGroup: "rbac.authorization.k8s.io"},
				},
			}
			fc := newFakeClient(rb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))

			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())
			Expect(emBinding.GetDisplayName()).To(Equal(bindingName))

			// Subject should be a Group (K8s subject kind is "Group").
			subj := emBinding.GetSubject()
			Expect(subj).NotTo(BeNil())
			Expect(subj.EffectiveKind()).To(Equal(iam.SubjectKindGroup))
			Expect(subj.EffectiveGroupID()).To(Equal(subjectID))

			// No finding since annotation is present.
			findingID := subjectFindingID(bindingID)
			Expect(b.GetModel().GetFindingById(findingID)).To(BeNil())
		})

		It("should set Identity subject when K8s subject kind is User", func() {
			identityID := uuid.New()
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: bindingNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: identityID.String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "viewer",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "alice", APIGroup: "rbac.authorization.k8s.io"},
				},
			}
			fc := newFakeClient(rb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())

			subj := emBinding.GetSubject()
			Expect(subj).NotTo(BeNil())
			Expect(subj.EffectiveKind()).To(Equal(iam.SubjectKindIdentity))
			Expect(subj.EffectiveIdentityID()).To(Equal(identityID))
		})

		It("should generate a finding when subject annotation is missing", func() {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: bindingNamespace,
					UID:       types.UID(uuid.New().String()),
					// No AnnotationSubjectID
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "admin",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "admins", APIGroup: "rbac.authorization.k8s.io"},
				},
			}
			fc := newFakeClient(rb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))

			findingID := subjectFindingID(bindingID)
			f := b.GetModel().GetFindingById(findingID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(MissingSubjectAnnotationFindingTypeID))
		})

		It("should resolve role reference through the name index", func() {
			roleID := uuid.New()
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: bindingNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: uuid.New().String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "indexed-role",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "team", APIGroup: "rbac.authorization.k8s.io"},
				},
			}
			fc := newFakeClient(rb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			// Pre-populate the index as if the Role controller already reconciled it.
			idx.Put(KindRole, bindingNamespace+"/indexed-role", roleID)

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())
			Expect(emBinding.GetRole()).NotTo(BeNil())
			Expect(emBinding.GetRole().EffectiveRoleID()).To(Equal(roleID))
		})

		It("should remove binding and finding on delete", func() {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: bindingNamespace,
					UID:       types.UID(uuid.New().String()),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "admin",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "admins", APIGroup: "rbac.authorization.k8s.io"},
				},
			}
			fc := newFakeClient(rb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")

			// Reconcile create.
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))
			findingID := subjectFindingID(bindingID)
			Expect(b.GetModel().GetFindingById(findingID)).NotTo(BeNil())

			// Delete and reconcile.
			Expect(fc.Delete(ctx, rb)).To(Succeed())
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(idx.Get(KindBinding, namespacedName.String())).To(Equal(uuid.Nil))
			Expect(b.GetModel().GetBindingById(bindingID)).To(BeNil())
			Expect(b.GetModel().GetFindingById(findingID)).To(BeNil())
		})
	})

	Context("When reconciling a ClusterRoleBinding", func() {
		const crbName = "test-cluster-rolebinding"
		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: crbName}

		It("should map a ClusterRoleBinding to an EmELand Binding", func() {
			crb := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: crbName,
					UID:  types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: uuid.New().String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "ServiceAccount", Name: "default", Namespace: "kube-system"},
				},
			}
			fc := newFakeClient(crb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.ClusterRoleBinding{}, "ClusterRoleBinding")

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, namespacedName.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))

			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())

			// ServiceAccount maps to Identity.
			subj := emBinding.GetSubject()
			Expect(subj).NotTo(BeNil())
			Expect(subj.EffectiveKind()).To(Equal(iam.SubjectKindIdentity))
		})
	})

	Context("When a Role arrives after its RoleBinding (late-arriving role)", func() {
		ctx := context.Background()

		It("should register a pending binding and resolve it when the Role appears", func() {
			roleName := "late-role"
			roleNamespace := "default"
			roleNN := types.NamespacedName{Name: roleName, Namespace: roleNamespace}

			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "binding-for-late-role",
					Namespace: roleNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: uuid.New().String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     roleName,
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "team", APIGroup: "rbac.authorization.k8s.io"},
				},
			}

			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: roleNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationRoleSpecID: uuid.New().String(),
					},
				},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
				},
			}

			fc := newFakeClient(rb, role)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			bindingReconciler := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.RoleBinding{}, "RoleBinding")
			roleReconciler := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.Role{}, "Role")
			roleReconciler.SetBindingReconciler(bindingReconciler)

			// Reconcile the binding BEFORE the role exists in the index.
			bindingNN := types.NamespacedName{Name: "binding-for-late-role", Namespace: roleNamespace}
			_, err = bindingReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: bindingNN})
			Expect(err).NotTo(HaveOccurred())

			// Binding should exist but without a role ref.
			bindingID := idx.Get(KindBinding, bindingNN.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))
			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())
			Expect(emBinding.GetRole()).To(BeNil())

			// Now reconcile the role - this should trigger pending binding resolution.
			_, err = roleReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: roleNN})
			Expect(err).NotTo(HaveOccurred())

			roleID := idx.Get(KindRole, roleIndexKey(roleNN))
			Expect(roleID).NotTo(Equal(uuid.Nil))

			// Drain the requeue channel and re-reconcile the binding.
			select {
			case evt := <-bindingReconciler.requeue:
				requeuedNN := types.NamespacedName{
					Name:      evt.Object.GetName(),
					Namespace: evt.Object.GetNamespace(),
				}
				_, err = bindingReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: requeuedNN})
				Expect(err).NotTo(HaveOccurred())
			default:
				Fail("expected a requeue event but channel was empty")
			}

			// Now the binding should have the role ref set.
			emBinding = b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())
			Expect(emBinding.GetRole()).NotTo(BeNil())
			Expect(emBinding.GetRole().EffectiveRoleID()).To(Equal(roleID))
		})
	})
})
