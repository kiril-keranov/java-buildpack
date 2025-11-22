package finalize

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/java-buildpack/src/java/containers"
	"github.com/cloudfoundry/java-buildpack/src/java/frameworks"
	"github.com/cloudfoundry/java-buildpack/src/java/jres"
	"github.com/cloudfoundry/libbuildpack"
)

type Finalizer struct {
	Stager    *libbuildpack.Stager
	Manifest  *libbuildpack.Manifest
	Installer *libbuildpack.Installer
	Log       *libbuildpack.Logger
	Command   *libbuildpack.Command
	Container containers.Container
}

// Run performs the finalize phase
func Run(f *Finalizer) error {
	f.Log.BeginStep("Finalizing Java")

	// Create container context
	ctx := &containers.Context{
		Stager:    f.Stager,
		Manifest:  f.Manifest,
		Installer: f.Installer,
		Log:       f.Log,
		Command:   f.Command,
	}

	// Create and populate container registry
	registry := containers.NewRegistry(ctx)
	registry.Register(containers.NewSpringBootContainer(ctx))
	registry.Register(containers.NewTomcatContainer(ctx))
	registry.Register(containers.NewGroovyContainer(ctx))
	registry.Register(containers.NewPlayContainer(ctx))
	registry.Register(containers.NewDistZipContainer(ctx))
	registry.Register(containers.NewJavaMainContainer(ctx))

	// Detect which container was used (should match supply phase)
	container, containerName, err := registry.Detect()
	if err != nil {
		f.Log.Error("Failed to detect container: %s", err.Error())
		return err
	}
	if container == nil {
		f.Log.Error("No suitable container found for this application")
		return fmt.Errorf("no suitable container found")
	}

	f.Log.Info("Finalizing container: %s", containerName)
	f.Container = container

	// Finalize JRE (memory calculator, jvmkill, etc.)
	if err := f.finalizeJRE(); err != nil {
		f.Log.Error("Failed to finalize JRE: %s", err.Error())
		return err
	}

	// Finalize frameworks (APM agents, etc.)
	if err := f.finalizeFrameworks(); err != nil {
		f.Log.Error("Failed to finalize frameworks: %s", err.Error())
		return err
	}

	// Call container's finalize method
	if err := container.Finalize(); err != nil {
		f.Log.Error("Failed to finalize container: %s", err.Error())
		return err
	}

	// Generate startup script
	if err := f.generateStartupScript(container); err != nil {
		f.Log.Error("Failed to generate startup script: %s", err.Error())
		return err
	}

	f.Log.Info("Java buildpack finalization complete")
	return nil
}

// finalizeJRE finalizes the JRE configuration (memory calculator, jvmkill, etc.)
func (f *Finalizer) finalizeJRE() error {
	f.Log.BeginStep("Finalizing JRE")

	// Create JRE context
	ctx := &jres.Context{
		Stager:    f.Stager,
		Manifest:  f.Manifest,
		Installer: f.Installer,
		Log:       f.Log,
		Command:   f.Command,
	}

	// Create and populate JRE registry
	registry := jres.NewRegistry(ctx)

	// Register the same JRE providers as in supply phase
	// We need to detect which one was used during supply
	registry.Register(jres.NewOpenJDKJRE(ctx))
	// Additional JRE providers:
	// registry.Register(jres.NewZuluJRE(ctx))
	// registry.Register(jres.NewGraalVMJRE(ctx))

	// Detect which JRE was installed (should match supply phase)
	jre, jreName, err := registry.Detect()
	if err != nil {
		f.Log.Error("Failed to detect JRE: %s", err.Error())
		return err
	}
	if jre == nil {
		f.Log.Warning("No JRE found during finalize, skipping JRE finalization")
		return nil
	}

	f.Log.Info("Finalizing JRE: %s", jreName)

	// Call JRE finalize (this will finalize memory calculator, jvmkill, etc.)
	if err := jre.Finalize(); err != nil {
		f.Log.Warning("Failed to finalize JRE: %s (continuing)", err.Error())
		// Don't fail the build if JRE finalization fails
		return nil
	}

	f.Log.Info("JRE finalization complete")
	return nil
}

// finalizeFrameworks finalizes framework components (APM agents, etc.)
func (f *Finalizer) finalizeFrameworks() error {
	f.Log.BeginStep("Finalizing frameworks")

	// Create framework context
	ctx := &frameworks.Context{
		Stager:    f.Stager,
		Manifest:  f.Manifest,
		Installer: f.Installer,
		Log:       f.Log,
		Command:   f.Command,
	}

	// Create and populate framework registry
	registry := frameworks.NewRegistry(ctx)
	registry.Register(frameworks.NewNewRelicFramework(ctx))
	registry.Register(frameworks.NewAppDynamicsFramework(ctx))
	registry.Register(frameworks.NewDynatraceFramework(ctx))

	// Detect all frameworks that were installed
	detectedFrameworks, frameworkNames, err := registry.DetectAll()
	if err != nil {
		f.Log.Warning("Failed to detect frameworks: %s", err.Error())
		return nil // Don't fail the build if framework detection fails
	}

	if len(detectedFrameworks) == 0 {
		f.Log.Info("No frameworks to finalize")
		return nil
	}

	f.Log.Info("Finalizing frameworks: %v", frameworkNames)

	// Finalize all detected frameworks
	for i, framework := range detectedFrameworks {
		f.Log.Info("Finalizing framework: %s", frameworkNames[i])
		if err := framework.Finalize(); err != nil {
			f.Log.Warning("Failed to finalize framework %s: %s", frameworkNames[i], err.Error())
			// Continue with other frameworks even if one fails
			continue
		}
	}

	return nil
}

// generateStartupScript creates the main startup script that Cloud Foundry will execute
func (f *Finalizer) generateStartupScript(container containers.Container) error {
	f.Log.BeginStep("Generating startup script")

	// Create .java-buildpack directory in HOME
	// In Cloud Foundry, $HOME is the app directory at runtime
	javaBuildpackDir := filepath.Join(f.Stager.BuildDir(), ".java-buildpack")
	if err := os.MkdirAll(javaBuildpackDir, 0755); err != nil {
		return fmt.Errorf("failed to create .java-buildpack directory: %w", err)
	}

	startScript := filepath.Join(javaBuildpackDir, "start.sh")

	// Get the container's startup command
	containerCommand, err := container.Release()
	if err != nil {
		return fmt.Errorf("failed to get container command: %w", err)
	}

	// Build startup script content
	scriptContent := fmt.Sprintf(`#!/bin/bash
set -e

# Source profile.d scripts (sets JAVA_HOME, etc.)
for script in $HOME/.profile.d/*.sh; do
  [ -r "$script" ] && source "$script"
done

# Source memory calculator script
if [ -r $DEPS_DIR/0/bin/memory_calculator.sh ]; then
  source $DEPS_DIR/0/bin/memory_calculator.sh
fi

# Source environment variables
if [ -d $DEPS_DIR/0/env ]; then
  for envfile in $DEPS_DIR/0/env/*; do
    [ -r "$envfile" ] && export $(basename "$envfile")="$(cat "$envfile")"
  done
fi

# Execute application
cd $HOME
exec %s
`, containerCommand)

	// Write startup script
	if err := os.WriteFile(startScript, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write startup script: %w", err)
	}

	f.Log.Info("Startup script generated: %s", startScript)
	return nil
}
