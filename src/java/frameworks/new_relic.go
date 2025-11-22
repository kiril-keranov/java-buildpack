package frameworks

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

// NewRelicFramework implements New Relic APM agent support
type NewRelicFramework struct {
	context *Context
}

// NewNewRelicFramework creates a new New Relic framework instance
func NewNewRelicFramework(ctx *Context) *NewRelicFramework {
	return &NewRelicFramework{context: ctx}
}

// Detect checks if New Relic should be included
func (n *NewRelicFramework) Detect() (string, error) {
	// Check for New Relic service binding
	vcapServices, err := GetVCAPServices()
	if err != nil {
		n.context.Log.Warning("Failed to parse VCAP_SERVICES: %s", err.Error())
		return "", nil
	}

	// New Relic can be bound as:
	// - "newrelic" service
	// - Services with "newrelic" tag
	if vcapServices.HasService("newrelic") || vcapServices.HasTag("newrelic") {
		return "New Relic Agent", nil
	}

	// Also check for NEW_RELIC_LICENSE_KEY environment variable
	if n.context.Stager.LinkDirectoryInDepDir(filepath.Join(n.context.Stager.BuildDir(), ".new-relic-credentials"), "new-relic-credentials") == nil {
		return "New Relic Agent", nil
	}

	return "", nil
}

// Supply installs the New Relic agent
func (n *NewRelicFramework) Supply() error {
	n.context.Log.BeginStep("Installing New Relic Agent")

	// Get New Relic agent dependency from manifest
	dep, err := n.context.Manifest.DefaultVersion("newrelic")
	if err != nil {
		n.context.Log.Warning("Unable to determine New Relic version, using default")
		dep = libbuildpack.Dependency{
			Name:    "newrelic",
			Version: "8.14.0", // Fallback version
		}
	}

	// Install New Relic agent JAR
	agentDir := filepath.Join(n.context.Stager.DepDir(), "new_relic_agent")
	if err := n.context.Installer.InstallDependency(dep, agentDir); err != nil {
		return fmt.Errorf("failed to install New Relic agent: %w", err)
	}

	// Find the New Relic agent JAR
	agentJar := filepath.Join(agentDir, "newrelic.jar")

	// Add javaagent to JAVA_OPTS
	javaOpts := fmt.Sprintf("-javaagent:%s", agentJar)

	// Get New Relic configuration from service binding
	vcapServices, _ := GetVCAPServices()
	service := vcapServices.GetService("newrelic")

	if service != nil {
		// Add license key from service credentials
		if licenseKey, ok := service.Credentials["licenseKey"].(string); ok && licenseKey != "" {
			javaOpts += fmt.Sprintf(" -Dnewrelic.config.license_key=%s", licenseKey)
		}

		// Add app name from service name
		if service.Name != "" {
			javaOpts += fmt.Sprintf(" -Dnewrelic.config.app_name='%s'", service.Name)
		}
	}

	// Write JAVA_OPTS to environment
	if err := n.context.Stager.WriteEnvFile("JAVA_OPTS", javaOpts); err != nil {
		return fmt.Errorf("failed to set JAVA_OPTS for New Relic: %w", err)
	}

	n.context.Log.Info("Installed New Relic Agent version %s", dep.Version)
	return nil
}

// Finalize performs final New Relic configuration
func (n *NewRelicFramework) Finalize() error {
	// New Relic doesn't require finalization
	return nil
}
