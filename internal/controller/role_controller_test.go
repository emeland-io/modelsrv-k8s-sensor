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
)

var _ = Describe("Role Controller", func() {
	Context("When reconciling a Role", func() {
		const roleName = "test-role"
		const roleNamespace = "default"
		roleSpecID := uuid.MustParse("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: roleName, Namespace: roleNamespace}

		It("should map a Role with RoleSpec annotation to an EmELand Role", func() {
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: roleNamespace,
					UID:       types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationRoleSpecID: roleSpecID.String(),
					},
				},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
				},
			}
			fc := newFakeClient(role)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.Role{}, "Role", nil)

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify the role was added to the model.
			roleID := idx.Get(KindRole, roleIndexKey(namespacedName))
			Expect(roleID).NotTo(Equal(uuid.Nil))

			emRole := b.GetModel().GetRoleById(roleID)
			Expect(emRole).NotTo(BeNil())
			Expect(emRole.GetDisplayName()).To(Equal(roleName))
			Expect(emRole.GetRoleSpecId()).To(Equal(roleSpecID))

			// No finding should exist since annotation is present.
			findingID := roleSpecFindingID(roleID)
			f := b.GetModel().GetFindingById(findingID)
			Expect(f).To(BeNil())
		})

		It("should generate a finding when RoleSpec annotation is missing", func() {
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: roleNamespace,
					UID:       types.UID(uuid.New().String()),
					// No AnnotationRoleSpecID
				},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
				},
			}
			fc := newFakeClient(role)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.Role{}, "Role", nil)

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			roleID := idx.Get(KindRole, roleIndexKey(namespacedName))
			Expect(roleID).NotTo(Equal(uuid.Nil))

			// Finding should exist.
			findingID := roleSpecFindingID(roleID)
			f := b.GetModel().GetFindingById(findingID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(MissingRoleSpecAnnotationFindingTypeID))
		})

		It("should remove role and finding on delete", func() {
			roleUID := uuid.New()
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: roleNamespace,
					UID:       types.UID(roleUID.String()),
				},
			}
			fc := newFakeClient(role)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.Role{}, "Role", nil)

			// Reconcile create.
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			roleID := idx.Get(KindRole, roleIndexKey(namespacedName))
			Expect(roleID).NotTo(Equal(uuid.Nil))

			// Ensure finding was created (no annotation).
			findingID := roleSpecFindingID(roleID)
			f := b.GetModel().GetFindingById(findingID)
			Expect(f).NotTo(BeNil())

			// Delete the K8s object and reconcile again.
			Expect(fc.Delete(ctx, role)).To(Succeed())
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Role and finding should be gone.
			Expect(idx.Get(KindRole, roleIndexKey(namespacedName))).To(Equal(uuid.Nil))
			Expect(b.GetModel().GetRoleById(roleID)).To(BeNil())
			Expect(b.GetModel().GetFindingById(findingID)).To(BeNil())
		})
	})

	Context("When reconciling a ClusterRole", func() {
		const clusterRoleName = "test-cluster-role"

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: clusterRoleName}

		It("should map a ClusterRole to an EmELand Role", func() {
			cr := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleName,
					UID:  types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationRoleSpecID: uuid.New().String(),
					},
				},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
				},
			}
			fc := newFakeClient(cr)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			r := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.ClusterRole{}, "ClusterRole", nil)

			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			roleID := idx.Get(KindRole, roleIndexKey(namespacedName))
			Expect(roleID).NotTo(Equal(uuid.Nil))
			Expect(b.GetModel().GetRoleById(roleID)).NotTo(BeNil())
		})

		It("should be resolvable by a ClusterRoleBinding via the name index", func() {
			cr := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleName,
					UID:  types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationRoleSpecID: uuid.New().String(),
					},
				},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
				},
			}
			crb := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-crb",
					UID:  types.UID(uuid.New().String()),
					Annotations: map[string]string{
						AnnotationSubjectID: uuid.New().String(),
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     clusterRoleName,
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "admins", APIGroup: "rbac.authorization.k8s.io"},
				},
			}

			fc := newFakeClient(cr, crb)
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			// Reconcile the ClusterRole first.
			roleReconciler := NewRoleReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.ClusterRole{}, "ClusterRole", nil)
			_, err = roleReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			roleID := idx.Get(KindRole, roleIndexKey(namespacedName))
			Expect(roleID).NotTo(Equal(uuid.Nil))

			// Now reconcile the ClusterRoleBinding - it should resolve the role ref.
			crbNN := types.NamespacedName{Name: "test-crb"}
			bindingReconciler := NewRoleBindingReconciler(fc, testScheme, b.GetModel(), idx, &rbacv1.ClusterRoleBinding{}, "ClusterRoleBinding", nil)
			_, err = bindingReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: crbNN})
			Expect(err).NotTo(HaveOccurred())

			bindingID := idx.Get(KindBinding, crbNN.String())
			Expect(bindingID).NotTo(Equal(uuid.Nil))

			emBinding := b.GetModel().GetBindingById(bindingID)
			Expect(emBinding).NotTo(BeNil())
			Expect(emBinding.GetRole()).NotTo(BeNil())
			Expect(emBinding.GetRole().EffectiveRoleID()).To(Equal(roleID))
		})
	})
})
