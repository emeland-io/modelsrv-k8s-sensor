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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

var testScheme *runtime.Scheme
var k8sClient client.Client
var ctx context.Context

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	testScheme = runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(testScheme)).To(Succeed())
	Expect(structurev1alpha1.AddToScheme(testScheme)).To(Succeed())
	Expect(corev1.AddToScheme(testScheme)).To(Succeed())
	Expect(appsv1.AddToScheme(testScheme)).To(Succeed())
	Expect(batchv1.AddToScheme(testScheme)).To(Succeed())
	Expect(networkingv1.AddToScheme(testScheme)).To(Succeed())

	// Global fake client for existing CRD controller tests.
	k8sClient = newFakeClient()
	ctx = context.Background()
})

// newFakeClient creates a fake K8s client pre-loaded with the given objects.
func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objs...).
		WithStatusSubresource(
			&structurev1alpha1.System{},
			&structurev1alpha1.API{},
			&structurev1alpha1.Component{},
			&structurev1alpha1.SystemInstance{},
		).
		Build()
}
