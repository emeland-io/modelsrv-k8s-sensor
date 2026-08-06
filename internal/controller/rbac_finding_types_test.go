package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/backend"
)

var _ = Describe("RBAC FindingTypes", func() {
	It("should register both well-known finding types", func() {
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		m := b.GetModel()

		err = RegisterRBACFindingTypes(m)
		Expect(err).NotTo(HaveOccurred())

		// MissingRoleSpecAnnotation type should exist.
		ft1 := m.GetFindingTypeById(MissingRoleSpecAnnotationFindingTypeID)
		Expect(ft1).NotTo(BeNil())
		Expect(ft1.GetDisplayName()).To(Equal("Missing RoleSpec Annotation"))

		// MissingSubjectAnnotation type should exist.
		ft2 := m.GetFindingTypeById(MissingSubjectAnnotationFindingTypeID)
		Expect(ft2).NotTo(BeNil())
		Expect(ft2.GetDisplayName()).To(Equal("Missing Subject Annotation"))
	})

	It("should be idempotent (calling twice does not error)", func() {
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		m := b.GetModel()

		err = RegisterRBACFindingTypes(m)
		Expect(err).NotTo(HaveOccurred())
		err = RegisterRBACFindingTypes(m)
		Expect(err).NotTo(HaveOccurred())
	})
})
