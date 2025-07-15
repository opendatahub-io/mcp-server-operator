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
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opendatahub-io/mcp-server-operator/test/utils"
)

// namespace where the project is deployed in
const (
	namespace = "mcp-server-operator-system"
	crName    = "mcp-server-test"
)

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
		By("undeploying the controller-manager")
		cmd := exec.Command("make", "undeploy")
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
		It("should successfully reconcile MCPServer CR and expose a working route", func() {
			// Create the MCPServer using the following YAML
			By("creating an MCPServer CR")
			mcpServerCR := fmt.Sprintf(`
apiVersion: mcpserver.opendatahub.io/v1
kind: MCPServer
metadata:
  name: %s
  namespace: %s
spec:
  image: "quay.io/rh-ee-cmclaugh/ocp-mcp-server:latest"
`, crName, namespace)

			// Apply the CR to the cluster, check if an error occurs.
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mcpServerCR)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create MCPServer CR")

			By("waiting until the MCPServer CR's overall condition is set to True")
			Eventually(func(g Gomega) {
				// Get the status condition, check if it's available, return error if there is one.
				jsonPath := `jsonpath={.status.conditions[?(@.type=="Available")].status}`
				cmd := exec.Command("kubectl", "get", "mcpserver", crName, "-n", namespace, "-o", jsonPath)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(strings.TrimSpace(output)).To(Equal("True"))
			}).Should(Succeed(), "MCPServer CR status did not become True")

			By("querying the route URL and verifying that the output is as expected")
			var routeHost, routePath string
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "route", crName, "-n", namespace, "-o", "jsonpath={.spec.host} {.spec.path}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).NotTo(BeEmpty(), "Route host and path should not be empty")

				// The output will be in the format "host /path", so we split it by the space.
				parts := strings.Split(strings.TrimSpace(output), " ")
				g.Expect(parts).To(HaveLen(2), "Expected output to contain both a host and a path")

				routeHost = parts[0]
				routePath = parts[1]
				g.Expect(routeHost).NotTo(BeEmpty())
				g.Expect(routePath).NotTo(BeEmpty())
			}).Should(Succeed(), "Should be able to get the route hostname and path")

			// Create the route URL using the host and the sse path
			routeURL := fmt.Sprintf("http://%s%s", routeHost, routePath)
			_, _ = fmt.Fprintf(GinkgoWriter, "Querying route URL: %s\n", routeURL)

			Eventually(func(g Gomega) {
				client := http.Client{
					Timeout: 15 * time.Second,
				}
				// Establish an HTTP Get request to the route's URL, create a response body
				resp, err := client.Get(routeURL)
				g.Expect(err).NotTo(HaveOccurred())

				// Close response body
				defer func() {
					err := resp.Body.Close()
					g.Expect(err).NotTo(HaveOccurred())
				}()

				g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
				g.Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("text/event-stream"))

				reader := bufio.NewReader(resp.Body)
				buffer := make([]byte, 1024)
				response, err := reader.Read(buffer)
				g.Expect(err).To(Or(BeNil(), Equal(io.EOF)))
				responseString := string(buffer[:response])

				expectedPattern := `event: endpoint\ndata: /message\?sessionId=.+`
				g.Expect(responseString).To(MatchRegexp(expectedPattern), "Response should match expected SSE format")

			}).Should(Succeed(), "The route should be available and respond correctly")
		})
		// +kubebuilder:scaffold:e2e-webhooks-checks

	})
})
