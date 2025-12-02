package integration_test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/switchblade"
	"github.com/cloudfoundry/switchblade/matchers"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testTomcat(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually
			name       string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			if name != "" {
				Expect(platform.Delete.Execute(name)).To(Succeed())
			}
		})

		context("with a simple servlet app", func() {
			it("successfully deploys and runs", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{
						"BP_JAVA_VERSION": "11",
					}).
					Execute(name, filepath.Join(fixtures, "container_tomcat"))

				// Print staging logs for debugging
				t.Logf("\n=== STAGING LOGS ===\n%s\n=== END STAGING LOGS ===\n", logs.String())

				Expect(err).NotTo(HaveOccurred(), logs.String)

				// Debug: Print deployment information
				t.Logf("Deployment Name: %s", deployment.Name)
				t.Logf("External URL: %s", deployment.ExternalURL)
				t.Logf("Internal URL: %s", deployment.InternalURL)

				// Get runtime logs from Docker container
				t.Logf("\n=== Fetching runtime logs from container ===")
				// Sleep briefly to allow container to start
				time.Sleep(2 * time.Second)

				// Use docker CLI to get logs
				cmd := exec.Command("docker", "logs", deployment.Name)
				runtimeLogs, err := cmd.CombinedOutput()
				if err != nil {
					t.Logf("Failed to get docker logs: %v", err)
				} else {
					t.Logf("\n=== RUNTIME LOGS ===\n%s\n=== END RUNTIME LOGS ===\n", string(runtimeLogs))
				}

				Eventually(deployment).Should(matchers.Serve(ContainSubstring("OK")))
			})
		})

		context("with JRE selection", func() {
			it("deploys with Java 8", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{
						"BP_JAVA_VERSION": "8",
					}).
					Execute(name, filepath.Join(fixtures, "container_tomcat"))
				Expect(err).NotTo(HaveOccurred(), logs.String)

				Expect(logs.String()).To(ContainSubstring("OpenJDK"))
				Eventually(deployment).Should(matchers.Serve(Not(BeEmpty())))
			})

			it("deploys with Java 11", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{
						"BP_JAVA_VERSION": "11",
					}).
					Execute(name, filepath.Join(fixtures, "container_tomcat"))
				Expect(err).NotTo(HaveOccurred(), logs.String)

				Expect(logs.String()).To(ContainSubstring("OpenJDK"))
				Eventually(deployment).Should(matchers.Serve(Not(BeEmpty())))
			})

			it("deploys with Java 17", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{
						"BP_JAVA_VERSION": "17",
					}).
					Execute(name, filepath.Join(fixtures, "container_tomcat"))
				Expect(err).NotTo(HaveOccurred(), logs.String)

				Expect(logs.String()).To(ContainSubstring("OpenJDK"))
				Eventually(deployment).Should(matchers.Serve(Not(BeEmpty())))
			})
		})

		context("with memory limits", func() {
			it("respects memory calculator settings", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{
						"BP_JAVA_VERSION":         "11",
						"JAVA_OPTS":               "-Xmx256m",
						"JBP_CONFIG_OPEN_JDK_JRE": "{jre: {version: 11.+}}",
					}).
					Execute(name, filepath.Join(fixtures, "container_tomcat"))

				Expect(err).NotTo(HaveOccurred(), logs.String)

				Expect(logs.String()).To(ContainSubstring("memory"))
				Eventually(deployment).Should(matchers.Serve(Not(BeEmpty())))
			})
		})
	}
}
