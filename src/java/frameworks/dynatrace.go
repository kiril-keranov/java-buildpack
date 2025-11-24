package frameworks

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

// DynatraceFramework implements Dynatrace OneAgent support
type DynatraceFramework struct {
	context *Context
}

// NewDynatraceFramework creates a new Dynatrace framework instance
func NewDynatraceFramework(ctx *Context) *DynatraceFramework {
	return &DynatraceFramework{context: ctx}
}

// Detect checks if Dynatrace should be included
func (d *DynatraceFramework) Detect() (string, error) {
	// Check for Dynatrace service binding
	vcapServices, err := GetVCAPServices()
	if err != nil {
		d.context.Log.Warning("Failed to parse VCAP_SERVICES: %s", err.Error())
		return "", nil
	}

	// Dynatrace can be bound as:
	// - "dynatrace" service (marketplace or label)
	// - Services with "dynatrace" tag
	// - User-provided services with "dynatrace" in the name (Docker platform)
	if vcapServices.HasService("dynatrace") || vcapServices.HasTag("dynatrace") || vcapServices.HasServiceByNamePattern("dynatrace") {
		return "Dynatrace OneAgent", nil
	}

	return "", nil
}

// Supply installs the Dynatrace agent
func (d *DynatraceFramework) Supply() error {
	d.context.Log.BeginStep("Installing Dynatrace OneAgent")

	// Get Dynatrace agent dependency from manifest
	dep, err := d.context.Manifest.DefaultVersion("dynatrace")
	if err != nil {
		d.context.Log.Warning("Unable to determine Dynatrace version, using default")
		dep = libbuildpack.Dependency{
			Name:    "dynatrace",
			Version: "1.283.0", // Fallback version
		}
	}

	// Install Dynatrace agent
	agentDir := filepath.Join(d.context.Stager.DepDir(), "dynatrace_one_agent")
	if err := d.context.Installer.InstallDependency(dep, agentDir); err != nil {
		return fmt.Errorf("failed to install Dynatrace agent: %w", err)
	}

	// Find the Dynatrace agent library
	agentLib := filepath.Join(agentDir, "agent", "lib64", "liboneagentproc.so")

	// Get Dynatrace configuration from service binding
	vcapServices, _ := GetVCAPServices()
	service := vcapServices.GetService("dynatrace")

	// If not found by label, try user-provided services (Docker platform)
	if service == nil {
		service = vcapServices.GetServiceByNamePattern("dynatrace")
	}

	// Build agentpath options
	javaOpts := fmt.Sprintf("-agentpath:%s", agentLib)

	if service != nil {
		// Add environment ID
		if envID, ok := service.Credentials["environmentid"].(string); ok && envID != "" {
			javaOpts += fmt.Sprintf("=environmentid=%s", envID)
		}

		// Add tenant token
		if token, ok := service.Credentials["apitoken"].(string); ok && token != "" {
			javaOpts += fmt.Sprintf(",tenanttoken=%s", token)
		}

		// Add API URL
		if apiURL, ok := service.Credentials["apiurl"].(string); ok && apiURL != "" {
			javaOpts += fmt.Sprintf(",server=%s", apiURL)
		}
	}

	// Write JAVA_OPTS to environment
	if err := d.context.Stager.WriteEnvFile("JAVA_OPTS", javaOpts); err != nil {
		return fmt.Errorf("failed to set JAVA_OPTS for Dynatrace: %w", err)
	}

	// Set LD_PRELOAD for Dynatrace
	ldPreload := filepath.Join(agentDir, "agent", "lib64", "liboneagentproc.so")
	if err := d.context.Stager.WriteEnvFile("LD_PRELOAD", ldPreload); err != nil {
		d.context.Log.Warning("Failed to set LD_PRELOAD for Dynatrace: %s", err.Error())
	}

	// Set DT_HOME
	dtHome := filepath.Join(agentDir, "agent")
	if err := d.context.Stager.WriteEnvFile("DT_HOME", dtHome); err != nil {
		d.context.Log.Warning("Failed to set DT_HOME for Dynatrace: %s", err.Error())
	}

	d.context.Log.Info("Installed Dynatrace OneAgent version %s", dep.Version)
	return nil
}

// Finalize performs final Dynatrace configuration
func (d *DynatraceFramework) Finalize() error {
	// Dynatrace doesn't require finalization
	return nil
}
