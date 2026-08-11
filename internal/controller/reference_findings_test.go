package controller

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/events"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

var _ = Describe("Reference findings", func() {

	Describe("APIInstanceReconciler", func() {
		var (
			bgCtx = context.Background()
			nn    = types.NamespacedName{Name: "api-svc", Namespace: "default"}
		)

		It("emits MissingResourceReference when api-reference annotation is absent", func() {
			uid := uuid.New()
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(svc)

			r := NewAPIInstanceReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &corev1.Service{}, "Service")
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, MissingResourceReference)
			f := b.GetModel().GetFindingById(fID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(finding.TypeIDForKind(MissingResourceReference)))
		})

		It("emits ReferencedResourceNotFound when api-reference points to unknown UUID", func() {
			uid := uuid.New()
			apiID := uuid.New()
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationAPIReference: apiID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(svc)

			r := NewAPIInstanceReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &corev1.Service{}, "Service")
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			f := b.GetModel().GetFindingById(fID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(finding.TypeIDForKind(ReferencedResourceNotFound)))

			// MissingResourceReference should NOT be set.
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
		})

		It("clears findings when api-reference points to a known API", func() {
			uid := uuid.New()
			apiID := uuid.New()
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationAPIReference: apiID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			// Pre-register the API in the model.
			api := mdlapi.NewAPI(apiID)
			api.SetDisplayName("Known API")
			Expect(b.GetModel().AddApi(api)).To(Succeed())

			fakeClient := newFakeClient(svc)
			r := NewAPIInstanceReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &corev1.Service{}, "Service")
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// No findings should exist.
			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(fID)).To(BeNil())
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
		})

		It("supports legacy apiId.emeland.io annotation", func() {
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

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(svc)

			r := NewAPIInstanceReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &corev1.Service{}, "Service")
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// Should emit ReferencedResourceNotFound (not MissingResourceReference).
			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(fID)).NotTo(BeNil())
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
		})

		It("clears findings when the resource is deleted", func() {
			uid := uuid.New()
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			idx := NewNameIndex()

			// Simulate a previous reconcile that left a finding.
			ai, id := apiInstanceFromMeta(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: nn.Name, Namespace: nn.Namespace, UID: types.UID(uid.String())},
			})
			Expect(b.GetModel().AddApiInstance(ai)).To(Succeed())
			idx.Put(KindAPIInstance, nn.String(), id)
			upsertMissingResourceReference(b.GetModel(), uid, events.APIInstanceResource, nn.Name, AnnotationAPIReference)

			fakeClient := newFakeClient() // resource gone
			r := NewAPIInstanceReconciler(fakeClient, testScheme, b.GetModel(), idx, &corev1.Service{}, "Service")
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(fID)).To(BeNil())
		})
	})

	Describe("WorkloadReconciler", func() {
		var (
			bgCtx = context.Background()
			nn    = types.NamespacedName{Name: "my-deploy", Namespace: "default"}
		)

		It("emits MissingResourceReference when component-reference annotation is absent", func() {
			uid := uuid.New()
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(deploy)

			r := NewWorkloadReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &appsv1.Deployment{}, "Deployment", nil)
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, MissingResourceReference)
			f := b.GetModel().GetFindingById(fID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(finding.TypeIDForKind(MissingResourceReference)))
		})

		It("emits ReferencedResourceNotFound when component-reference points to unknown UUID", func() {
			uid := uuid.New()
			compID := uuid.New()
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationComponentReference: compID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(deploy)

			r := NewWorkloadReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &appsv1.Deployment{}, "Deployment", nil)
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(fID)).NotTo(BeNil())
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
		})

		It("clears findings when component-reference points to a known Component", func() {
			uid := uuid.New()
			compID := uuid.New()
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
					UID:       types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationComponentReference: compID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			comp := component.NewComponent(compID)
			comp.SetDisplayName("Known Component")
			Expect(b.GetModel().AddComponent(comp)).To(Succeed())

			fakeClient := newFakeClient(deploy)
			r := NewWorkloadReconciler(fakeClient, testScheme, b.GetModel(), NewNameIndex(), &appsv1.Deployment{}, "Deployment", nil)
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(fID)).To(BeNil())
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
		})
	})

	Describe("NamespaceReconciler", func() {
		var (
			bgCtx = context.Background()
		)

		It("emits ReferencedResourceNotFound when context-parent points to unknown UUID", func() {
			uid := uuid.New()
			clusterUID := uuid.New()
			parentID := uuid.New()

			// Create kube-system namespace first to set cluster context.
			kubeSystemNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
					UID:  types.UID(clusterUID.String()),
				},
			}
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-ns",
					UID:  types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationContextParent: parentID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(kubeSystemNs, ns)
			idx := NewNameIndex()

			r := &NamespaceReconciler{
				Client:   fakeClient,
				Scheme:   testScheme,
				Model:    b.GetModel(),
				Index:    idx,
				RuleEval: nil,
			}

			// Reconcile kube-system first.
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
			Expect(err).NotTo(HaveOccurred())

			// Reconcile my-ns with context-parent pointing to unknown UUID.
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-ns"}})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			f := b.GetModel().GetFindingById(fID)
			Expect(f).NotTo(BeNil())
			Expect(f.GetFindingTypeId()).To(Equal(finding.TypeIDForKind(ReferencedResourceNotFound)))
		})

		It("clears finding when context-parent points to a known Context", func() {
			uid := uuid.New()
			clusterUID := uuid.New()
			parentID := uuid.New()

			kubeSystemNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
					UID:  types.UID(clusterUID.String()),
				},
			}
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app-ns",
					UID:  types.UID(uid.String()),
					Annotations: map[string]string{
						AnnotationContextParent: parentID.String(),
					},
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			// Pre-register the parent context.
			parentCtx := mdlctx.NewContext(parentID)
			parentCtx.SetDisplayName("Application Context")
			Expect(b.GetModel().AddContext(parentCtx)).To(Succeed())

			fakeClient := newFakeClient(kubeSystemNs, ns)
			idx := NewNameIndex()

			r := &NamespaceReconciler{
				Client:   fakeClient,
				Scheme:   testScheme,
				Model:    b.GetModel(),
				Index:    idx,
				RuleEval: nil,
			}

			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
			Expect(err).NotTo(HaveOccurred())
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "app-ns"}})
			Expect(err).NotTo(HaveOccurred())

			fID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(fID)).To(BeNil())
		})

		It("does not emit MissingResourceReference when context-parent is absent", func() {
			uid := uuid.New()
			clusterUID := uuid.New()

			kubeSystemNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
					UID:  types.UID(clusterUID.String()),
				},
			}
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "plain-ns",
					UID:  types.UID(uid.String()),
				},
			}

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			fakeClient := newFakeClient(kubeSystemNs, ns)
			idx := NewNameIndex()

			r := &NamespaceReconciler{
				Client:   fakeClient,
				Scheme:   testScheme,
				Model:    b.GetModel(),
				Index:    idx,
				RuleEval: nil,
			}

			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kube-system"}})
			Expect(err).NotTo(HaveOccurred())
			_, err = r.Reconcile(bgCtx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "plain-ns"}})
			Expect(err).NotTo(HaveOccurred())

			// context-parent is optional, no MissingResourceReference finding.
			missingID := referenceFindingID(uid, MissingResourceReference)
			Expect(b.GetModel().GetFindingById(missingID)).To(BeNil())
			danglingID := referenceFindingID(uid, ReferencedResourceNotFound)
			Expect(b.GetModel().GetFindingById(danglingID)).To(BeNil())
		})
	})

	Describe("RegisterReferenceFindingTypes", func() {
		It("registers both well-known finding types", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			Expect(RegisterReferenceFindingTypes(b.GetModel())).To(Succeed())

			ft1 := b.GetModel().GetFindingTypeById(finding.TypeIDForKind(ReferencedResourceNotFound))
			Expect(ft1).NotTo(BeNil())
			Expect(ft1.GetDisplayName()).To(Equal("ReferencedResourceNotFound"))

			ft2 := b.GetModel().GetFindingTypeById(finding.TypeIDForKind(MissingResourceReference))
			Expect(ft2).NotTo(BeNil())
			Expect(ft2.GetDisplayName()).To(Equal("MissingResourceReference"))
		})
	})
})
