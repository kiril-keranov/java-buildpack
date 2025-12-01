package jres

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

// OpenJDKJRE implements the JRE interface for OpenJDK
type OpenJDKJRE struct {
	ctx              *Context
	jreDir           string
	version          string
	javaHome         string
	memoryCalc       *MemoryCalculator
	jvmkill          *JVMKillAgent
	installedVersion string
}

// NewOpenJDKJRE creates a new OpenJDK JRE provider
func NewOpenJDKJRE(ctx *Context) *OpenJDKJRE {
	jreDir := filepath.Join(ctx.Stager.DepDir(), "jre")

	return &OpenJDKJRE{
		ctx:    ctx,
		jreDir: jreDir,
	}
}

// Name returns the name of this JRE provider
func (o *OpenJDKJRE) Name() string {
	return "OpenJDK"
}

// Detect returns true if OpenJDK should be used
// OpenJDK is the default JRE, so it always returns true unless another JRE is explicitly configured
func (o *OpenJDKJRE) Detect() (bool, error) {
	// Check if another JRE is explicitly configured via environment
	// For now, OpenJDK is always selected (default)
	// In the future, we'll check JBP_CONFIG_COMPONENTS environment variable
	return true, nil
}

// Supply installs the OpenJDK JRE and its components
func (o *OpenJDKJRE) Supply() error {
	o.ctx.Log.BeginStep("Installing OpenJDK JRE")

	// Determine version
	dep, err := GetJREVersion(o.ctx, "openjdk")
	if err != nil {
		o.ctx.Log.Warning("Unable to determine OpenJDK version from manifest, using default")
		// Fallback to hardcoded version
		dep = libbuildpack.Dependency{
			Name:    "openjdk",
			Version: "17.0.13",
		}
	}

	o.version = dep.Version
	o.ctx.Log.Info("Installing OpenJDK %s", o.version)

	// Install JRE
	if err := o.ctx.Installer.InstallDependency(dep, o.jreDir); err != nil {
		return fmt.Errorf("failed to install OpenJDK: %w", err)
	}

	// Find the actual JAVA_HOME (handle nested directories from tar extraction)
	javaHome, err := o.findJavaHome()
	if err != nil {
		return fmt.Errorf("failed to find JAVA_HOME: %w", err)
	}
	o.javaHome = javaHome
	o.installedVersion = o.version

	// Set up JAVA_HOME environment
	if err := SetupJavaHome(o.ctx, o.jreDir); err != nil {
		return fmt.Errorf("failed to set up JAVA_HOME: %w", err)
	}

	// Create profile.d script to export JAVA_HOME at runtime
	// This is needed for containers like DistZip that use startup scripts expecting $JAVA_HOME
	if err := o.writeProfileDScript(); err != nil {
		o.ctx.Log.Warning("Could not write java.sh profile.d script: %s", err.Error())
	} else {
		o.ctx.Log.Debug("Created profile.d script: java.sh")
	}

	// Determine Java major version
	javaMajorVersion, err := DetermineJavaVersion(javaHome)
	if err != nil {
		o.ctx.Log.Warning("Could not determine Java version: %s", err.Error())
		javaMajorVersion = 17 // default
	}
	o.ctx.Log.Info("Detected Java major version: %d", javaMajorVersion)

	// Install JVMKill agent
	o.jvmkill = NewJVMKillAgent(o.ctx, o.jreDir, o.version)
	if err := o.jvmkill.Supply(); err != nil {
		o.ctx.Log.Warning("Failed to install JVMKill agent: %s (continuing)", err.Error())
		// Non-fatal - continue without jvmkill
	}

	// Install Memory Calculator
	o.memoryCalc = NewMemoryCalculator(o.ctx, o.jreDir, o.version, javaMajorVersion)
	if err := o.memoryCalc.Supply(); err != nil {
		o.ctx.Log.Warning("Failed to install Memory Calculator: %s (continuing)", err.Error())
		// Non-fatal - continue without memory calculator
	}

	o.ctx.Log.Info("OpenJDK JRE installation complete")
	return nil
}

// Finalize performs final JRE configuration
func (o *OpenJDKJRE) Finalize() error {
	o.ctx.Log.BeginStep("Finalizing OpenJDK JRE configuration")

	// Find the actual JAVA_HOME (needed if finalize is called on a fresh instance)
	if o.javaHome == "" {
		javaHome, err := o.findJavaHome()
		if err != nil {
			o.ctx.Log.Warning("Failed to find JAVA_HOME: %s", err.Error())
		} else {
			o.javaHome = javaHome
		}
	}

	// Determine Java major version for memory calculator
	javaMajorVersion := 17 // default
	if o.javaHome != "" {
		if ver, err := DetermineJavaVersion(o.javaHome); err == nil {
			javaMajorVersion = ver
		}
	}

	// Reconstruct JVMKill agent component if not already set
	// This handles the case where finalize is called on a fresh instance
	if o.jvmkill == nil {
		o.jvmkill = NewJVMKillAgent(o.ctx, o.jreDir, o.version)
	}

	// Finalize JVMKill agent
	if err := o.jvmkill.Finalize(); err != nil {
		o.ctx.Log.Warning("Failed to finalize JVMKill agent: %s", err.Error())
		// Non-fatal
	}

	// Reconstruct Memory Calculator component if not already set
	if o.memoryCalc == nil {
		o.memoryCalc = NewMemoryCalculator(o.ctx, o.jreDir, o.version, javaMajorVersion)
	}

	// Finalize Memory Calculator
	if err := o.memoryCalc.Finalize(); err != nil {
		o.ctx.Log.Warning("Failed to finalize Memory Calculator: %s", err.Error())
		// Non-fatal
	}

	o.ctx.Log.Info("OpenJDK JRE finalization complete")
	return nil
}

// JavaHome returns the path to JAVA_HOME
func (o *OpenJDKJRE) JavaHome() string {
	return o.javaHome
}

// Version returns the installed JRE version
func (o *OpenJDKJRE) Version() string {
	return o.installedVersion
}

// findJavaHome locates the actual JAVA_HOME directory after extraction
// OpenJDK tarballs usually extract to jdk-* or jre-* subdirectories
func (o *OpenJDKJRE) findJavaHome() (string, error) {
	entries, err := os.ReadDir(o.jreDir)
	if err != nil {
		return "", fmt.Errorf("failed to read JRE directory: %w", err)
	}

	// Look for jdk-* or jre-* subdirectory
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Check for common OpenJDK directory patterns
			if len(name) > 3 && (name[:3] == "jdk" || name[:3] == "jre") {
				path := filepath.Join(o.jreDir, name)
				// Verify it has a bin directory with java
				if _, err := os.Stat(filepath.Join(path, "bin", "java")); err == nil {
					return path, nil
				}
			}
		}
	}

	// If no subdirectory found, check if jreDir itself is valid
	if _, err := os.Stat(filepath.Join(o.jreDir, "bin", "java")); err == nil {
		return o.jreDir, nil
	}

	return "", fmt.Errorf("could not find valid JAVA_HOME in %s", o.jreDir)
}

// writeProfileDScript creates a profile.d script that exports JAVA_HOME at runtime
// This is needed for containers like DistZip that use startup scripts expecting $JAVA_HOME
func (o *OpenJDKJRE) writeProfileDScript() error {
	// Determine the relative path from jreDir to javaHome
	relPath, err := filepath.Rel(o.jreDir, o.javaHome)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Build the JAVA_HOME path using $DEPS_DIR environment variable
	// This allows the path to work at runtime when the app is staged
	var javaHomePath string
	if relPath == "." {
		// JAVA_HOME is directly at jreDir
		javaHomePath = "$DEPS_DIR/0/jre"
	} else {
		// JAVA_HOME is in a subdirectory (e.g., jdk-17.0.13)
		javaHomePath = fmt.Sprintf("$DEPS_DIR/0/jre/%s", relPath)
	}

	// Create the profile.d script content
	envContent := fmt.Sprintf("export JAVA_HOME=%s\n", javaHomePath)

	// Write the profile.d script
	if err := o.ctx.Stager.WriteProfileD("java.sh", envContent); err != nil {
		return fmt.Errorf("failed to write profile.d script: %w", err)
	}

	return nil
}
