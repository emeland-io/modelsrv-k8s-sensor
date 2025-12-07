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

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gitlab.com/emeland/k8s-model/test/utils"
)

const emelandNamespace = "emeland-k8s-system"
const appNamespace = "test-app"

const apiBaseURL = "http://modelsrv.local/api"
const swaggerBaseURL = "http://modelsrv.local/swagger"

var _ = Describe("EmELand Model-Server for Kubernetes", Ordered, func() {
	BeforeAll(func() {
		By("installing prometheus operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed())

		By("installing the cert-manager")
		Expect(utils.InstallCertManager()).To(Succeed())

		// currently the cloud-provider-kind provides the gateway
		// By("installing the Traefik ingress controller")
		// Expect(utils.InstallTraefikIngressController()).To(Succeed())

		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", emelandNamespace)
		_, _ = utils.Run(cmd)

		By("creating the app namespace")
		cmd = exec.Command("kubectl", "create", "ns", appNamespace)
		_, _ = utils.Run(cmd)

	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		utils.UninstallPrometheusOperator()

		By("uninstalling the cert-manager bundle")
		utils.UninstallCertManager()

		// currently the cloud-provider-kind provides the gateway
		// By("uninstalling the Traefik ingress controller")
		// utils.UninstallTraefikIngressController()

		By("removing the app namespace")
		cmd := exec.Command("kubectl", "delete", "ns", appNamespace)
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", emelandNamespace)
		_, _ = utils.Run(cmd)

	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			// projectimage stores the name of the image used in the example
			var imageRepo = "example.com/emeland-k8s"
			var imageVersion = "v0.0.1"
			var projectImage = fmt.Sprintf("%s:%s", imageRepo, imageVersion)

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectImage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "HELM_CRD_RELEASE=e2e-test-crds", "helm-install-crds")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager, using helm")
			cmd = exec.Command("make", "HELM_RELEASE=e2e-test",
				fmt.Sprintf("IMAGE_REPO=%s", imageRepo),
				fmt.Sprintf("IMAGE_VERSION=%s", imageVersion),
				fmt.Sprintf("KUBE_NAMESPACE=%s", emelandNamespace),
				"helm-install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name

				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", emelandNamespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", emelandNamespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

		})
		It("should load the test application successfully", func() {
			By("successfully loading the small set of model resources")
			cmd := exec.Command("kubectl", "apply", "-f", "./test/e2e/fixtures/minimal-app.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
