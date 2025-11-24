package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testFrameworks(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect
			name   string
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

		context("APM Agents", func() {
			context("with New Relic service binding", func() {
				it("detects and installs New Relic agent", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"newrelic": {
								"licenseKey": "test-license-key-1234567890abcdef",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "11",
						}).
						Execute(name, filepath.Join(fixtures, "container_spring_boot_staged"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					// Verify New Relic agent was detected and installed
					Expect(logs.String()).To(ContainSubstring("New Relic Agent"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})

				it("configures New Relic with license key from service binding", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"my-newrelic-service": {
								"licenseKey": "abc123def456ghi789jkl012mno345pq",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "17",
						}).
						Execute(name, filepath.Join(fixtures, "integration_valid"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					Expect(logs.String()).To(ContainSubstring("New Relic Agent"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})
			})

			context("with AppDynamics service binding", func() {
				it("detects and installs AppDynamics agent", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"appdynamics": {
								"account-access-key": "test-access-key",
								"account-name":       "customer1",
								"host-name":          "appdynamics.example.com",
								"port":               "443",
								"ssl-enabled":        "true",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "11",
						}).
						Execute(name, filepath.Join(fixtures, "container_spring_boot_staged"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					// Verify AppDynamics agent was detected and installed
					Expect(logs.String()).To(ContainSubstring("AppDynamics Agent"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})

				it("configures AppDynamics with controller info from service binding", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"my-appdynamics-service": {
								"account-access-key": "xyz789",
								"account-name":       "production-account",
								"host-name":          "controller.appdynamics.example.com",
								"port":               "8090",
								"ssl-enabled":        "false",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "17",
						}).
						Execute(name, filepath.Join(fixtures, "integration_valid"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					Expect(logs.String()).To(ContainSubstring("AppDynamics Agent"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})
			})

			context("with Dynatrace service binding", func() {
				it("detects and installs Dynatrace agent", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"dynatrace": {
								"environmentid": "abc12345",
								"apitoken":      "test-api-token-xyz",
								"apiurl":        "https://abc12345.live.dynatrace.com/api",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "11",
						}).
						Execute(name, filepath.Join(fixtures, "container_spring_boot_staged"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					// Verify Dynatrace agent was detected and installed
					Expect(logs.String()).To(ContainSubstring("Dynatrace"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})

				it("configures Dynatrace with environment ID from service binding", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"my-dynatrace-service": {
								"environmentid": "xyz78901",
								"apitoken":      "dt0c01.XXXXXXXXX.YYYYYYYYYYYY",
								"apiurl":        "https://xyz78901.live.dynatrace.com/api",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "17",
						}).
						Execute(name, filepath.Join(fixtures, "integration_valid"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					Expect(logs.String()).To(ContainSubstring("Dynatrace"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})
			})

			context("with multiple APM agents", func() {
				it("can handle multiple agent service bindings", func() {
					deployment, logs, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							"newrelic": {
								"licenseKey": "test-license-key",
							},
							"appdynamics": {
								"account-access-key": "test-key",
								"account-name":       "test-account",
								"host-name":          "controller.appdynamics.com",
							},
						}).
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "11",
						}).
						Execute(name, filepath.Join(fixtures, "integration_valid"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					// Both agents should be detected
					Expect(logs.String()).To(Or(
						ContainSubstring("New Relic Agent"),
						ContainSubstring("AppDynamics Agent"),
					))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})
			})

			context("without APM service bindings", func() {
				it("does not install any APM agents", func() {
					deployment, logs, err := platform.Deploy.
						WithEnv(map[string]string{
							"BP_JAVA_VERSION": "11",
						}).
						Execute(name, filepath.Join(fixtures, "integration_valid"))
					Expect(err).NotTo(HaveOccurred(), logs.String)

					// No APM agents should be mentioned
					Expect(logs.String()).NotTo(ContainSubstring("New Relic Agent"))
					Expect(logs.String()).NotTo(ContainSubstring("AppDynamics Agent"))
					Expect(deployment.ExternalURL).NotTo(BeEmpty())
				})
			})
		})
	}
}
