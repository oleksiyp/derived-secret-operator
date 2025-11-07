//go:build e2e
// +build e2e

/*
Copyright 2025.

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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/oleksiyp/derived-secret-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "derived-secret-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "derived-secret-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "derived-secret-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "derived-secret-operator-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=derived-secret-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		It("should update derived secret when master password is switched", func() {
			const testNamespace = "default"
			const derivedSecretName = "test-password-switch"
			const firstMasterPasswordName = "mp-first"
			const secondMasterPasswordName = "mp-second"

			By("creating the first master password")
			firstMPYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: %s
spec:
  length: 86
`, firstMasterPasswordName)
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(firstMPYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create first master password")

			By("waiting for first master password to be ready")
			verifyFirstMPReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "masterpassword", firstMasterPasswordName,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyFirstMPReady, 30*time.Second).Should(Succeed())

			By("creating derived secret using first master password")
			derivedSecretYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: %s
  namespace: %s
spec:
  type: Opaque
  keys:
    password:
      type: password
      masterPassword: %s
`, derivedSecretName, testNamespace, firstMasterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(derivedSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create derived secret")

			By("waiting for derived secret to be ready")
			verifyDerivedSecretReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyDerivedSecretReady, 30*time.Second).Should(Succeed())

			By("getting the initial secret value")
			cmd = exec.Command("kubectl", "get", "secret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.data.password}")
			firstSecretValue, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial secret value")
			Expect(firstSecretValue).NotTo(BeEmpty(), "Initial secret value should not be empty")

			By("creating the second master password")
			secondMPYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: %s
spec:
  length: 86
`, secondMasterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(secondMPYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create second master password")

			By("waiting for second master password to be ready")
			verifySecondMPReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "masterpassword", secondMasterPasswordName,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifySecondMPReady, 30*time.Second).Should(Succeed())

			By("updating derived secret to use second master password")
			updatedDerivedSecretYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: %s
  namespace: %s
spec:
  type: Opaque
  keys:
    password:
      type: password
      masterPassword: %s
`, derivedSecretName, testNamespace, secondMasterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(updatedDerivedSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to update derived secret")

			By("waiting for secret value to change")
			verifySecretChanged := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.data.password}")
				newSecretValue, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newSecretValue).NotTo(BeEmpty())
				g.Expect(newSecretValue).NotTo(Equal(firstSecretValue),
					"Secret value should have changed after switching master password")
			}
			Eventually(verifySecretChanged, 30*time.Second).Should(Succeed())

			By("cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "derivedsecret", derivedSecretName, "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "masterpassword", firstMasterPasswordName)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "masterpassword", secondMasterPasswordName)
			_, _ = utils.Run(cmd)
		})

		It("should recreate DerivedSecret's secret when deleted", func() {
			const testNamespace = "default"
			const derivedSecretName = "test-secret-recreation"
			const masterPasswordName = "mp-recreation"

			By("creating master password")
			mpYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: %s
spec:
  length: 86
`, masterPasswordName)
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mpYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create master password")

			By("waiting for master password to be ready")
			Eventually(verifyMasterPasswordReady(masterPasswordName), 30*time.Second).Should(Succeed())

			By("creating derived secret")
			derivedSecretYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: %s
  namespace: %s
spec:
  type: Opaque
  keys:
    password:
      type: password
      masterPassword: %s
`, derivedSecretName, testNamespace, masterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(derivedSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create derived secret")

			By("waiting for derived secret to be ready")
			verifyDerivedSecretReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyDerivedSecretReady, 30*time.Second).Should(Succeed())

			By("getting the initial secret value and hash")
			cmd = exec.Command("kubectl", "get", "secret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.data.password}")
			initialSecretValue, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial secret value")
			Expect(initialSecretValue).NotTo(BeEmpty(), "Initial secret value should not be empty")

			cmd = exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.keyHashes.password}")
			initialHash, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial hash")
			Expect(initialHash).NotTo(BeEmpty(), "Initial hash should not be empty")

			By("deleting the secret")
			cmd = exec.Command("kubectl", "delete", "secret", derivedSecretName, "-n", testNamespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete secret")

			By("waiting for secret to be recreated")
			verifySecretRecreated := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.data.password}")
				recreatedSecretValue, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(recreatedSecretValue).NotTo(BeEmpty())
				g.Expect(recreatedSecretValue).To(Equal(initialSecretValue),
					"Secret value should be the same after recreation (deterministic)")
			}
			Eventually(verifySecretRecreated, 30*time.Second).Should(Succeed())

			By("verifying hash remains the same")
			cmd = exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.keyHashes.password}")
			recreatedHash, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(recreatedHash).To(Equal(initialHash), "Hash should remain the same after recreation")

			By("cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "derivedsecret", derivedSecretName, "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "masterpassword", masterPasswordName)
			_, _ = utils.Run(cmd)
		})

		It("should regenerate DerivedSecrets when MasterPassword secret changes", func() {
			const testNamespace = "default"
			const derivedSecretName = "test-mp-secret-change"
			const masterPasswordName = "mp-secret-change"

			By("creating master password")
			mpYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: %s
spec:
  length: 86
`, masterPasswordName)
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mpYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create master password")

			By("waiting for master password to be ready")
			Eventually(verifyMasterPasswordReady(masterPasswordName), 30*time.Second).Should(Succeed())

			By("creating derived secret")
			derivedSecretYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: %s
  namespace: %s
spec:
  type: Opaque
  keys:
    password:
      type: password
      masterPassword: %s
`, derivedSecretName, testNamespace, masterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(derivedSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create derived secret")

			By("waiting for derived secret to be ready")
			verifyDerivedSecretReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyDerivedSecretReady, 30*time.Second).Should(Succeed())

			By("getting the initial secret value and hash")
			cmd = exec.Command("kubectl", "get", "secret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.data.password}")
			initialSecretValue, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial secret value")
			Expect(initialSecretValue).NotTo(BeEmpty(), "Initial secret value should not be empty")

			cmd = exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.keyHashes.password}")
			initialHash, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial hash")
			Expect(initialHash).NotTo(BeEmpty(), "Initial hash should not be empty")

			By("changing the MasterPassword secret")
			mpSecretName := masterPasswordName + "-mp"
			newSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  masterPassword: "new-different-master-password-value-for-testing"
`, mpSecretName, namespace)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(newSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to update MasterPassword secret")

			By("waiting for derived secret value to change")
			verifySecretChanged := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.data.password}")
				newSecretValue, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newSecretValue).NotTo(BeEmpty())
				g.Expect(newSecretValue).NotTo(Equal(initialSecretValue),
					"Secret value should have changed after MasterPassword secret change")
			}
			Eventually(verifySecretChanged, 30*time.Second).Should(Succeed())

			By("verifying hash changed")
			verifyHashChanged := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.keyHashes.password}")
				newHash, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newHash).NotTo(BeEmpty())
				g.Expect(newHash).NotTo(Equal(initialHash),
					"Hash should have changed after MasterPassword secret change")
			}
			Eventually(verifyHashChanged, 30*time.Second).Should(Succeed())

			By("cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "derivedsecret", derivedSecretName, "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "masterpassword", masterPasswordName)
			_, _ = utils.Run(cmd)
		})

		It("should handle MasterPassword secret deletion gracefully", func() {
			const testNamespace = "default"
			const derivedSecretName = "test-mp-secret-deletion"
			const masterPasswordName = "mp-secret-deletion"

			By("creating master password")
			mpYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: %s
spec:
  length: 86
`, masterPasswordName)
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mpYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create master password")

			By("waiting for master password to be ready")
			Eventually(verifyMasterPasswordReady(masterPasswordName), 30*time.Second).Should(Succeed())

			By("creating derived secret")
			derivedSecretYAML := fmt.Sprintf(`
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: %s
  namespace: %s
spec:
  type: Opaque
  keys:
    password:
      type: password
      masterPassword: %s
`, derivedSecretName, testNamespace, masterPasswordName)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(derivedSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create derived secret")

			By("waiting for derived secret to be ready")
			verifyDerivedSecretReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyDerivedSecretReady, 30*time.Second).Should(Succeed())

			By("getting the initial secret value")
			cmd = exec.Command("kubectl", "get", "secret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.data.password}")
			initialSecretValue, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial secret value")
			Expect(initialSecretValue).NotTo(BeEmpty(), "Initial secret value should not be empty")

			By("deleting the MasterPassword secret")
			mpSecretName := masterPasswordName + "-mp"
			cmd = exec.Command("kubectl", "delete", "secret", mpSecretName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete MasterPassword secret")

			By("verifying DerivedSecret becomes not ready")
			verifyDerivedSecretNotReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "derivedsecret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.ready}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("false"), "DerivedSecret should not be ready when MasterPassword secret is missing")
			}
			Eventually(verifyDerivedSecretNotReady, 30*time.Second).Should(Succeed())

			By("verifying the derived secret remains unchanged")
			cmd = exec.Command("kubectl", "get", "secret", derivedSecretName,
				"-n", testNamespace,
				"-o", "jsonpath={.data.password}")
			unchangedSecretValue, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchangedSecretValue).To(Equal(initialSecretValue),
				"Derived secret value should remain unchanged when MasterPassword secret is deleted")

			By("recreating the MasterPassword secret with new value")
			newSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/managed-by: derived-secret-operator
type: Opaque
stringData:
  masterPassword: "recreated-master-password-value-for-testing"
`, mpSecretName, namespace)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(newSecretYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to recreate MasterPassword secret")

			By("waiting for derived secret to become ready again")
			Eventually(verifyDerivedSecretReady, 30*time.Second).Should(Succeed())

			By("verifying derived secret value changed with new master password")
			verifySecretChanged := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secret", derivedSecretName,
					"-n", testNamespace,
					"-o", "jsonpath={.data.password}")
				newSecretValue, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newSecretValue).NotTo(BeEmpty())
				g.Expect(newSecretValue).NotTo(Equal(initialSecretValue),
					"Secret value should have changed after MasterPassword secret recreation")
			}
			Eventually(verifySecretChanged, 30*time.Second).Should(Succeed())

			By("cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "derivedsecret", derivedSecretName, "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "masterpassword", masterPasswordName)
			_, _ = utils.Run(cmd)
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// verifyMasterPasswordReady returns a Gomega assertion function that checks if a MasterPassword is ready.
// This helper reduces duplication across E2E tests.
func verifyMasterPasswordReady(name string) func(Gomega) {
	return func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "masterpassword", name,
			"-o", "jsonpath={.status.ready}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("true"))
	}
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
