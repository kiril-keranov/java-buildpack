package containers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/java-buildpack/src/java/jres"
	"github.com/cloudfoundry/libbuildpack"
)

// TomcatContainer handles servlet/WAR applications
type TomcatContainer struct {
	context *Context
}

// NewTomcatContainer creates a new Tomcat container
func NewTomcatContainer(ctx *Context) *TomcatContainer {
	return &TomcatContainer{
		context: ctx,
	}
}

// Detect checks if this is a Tomcat/servlet application
func (t *TomcatContainer) Detect() (string, error) {
	buildDir := t.context.Stager.BuildDir()

	// Check for WEB-INF directory (exploded WAR)
	webInf := filepath.Join(buildDir, "WEB-INF")
	if _, err := os.Stat(webInf); err == nil {
		t.context.Log.Debug("Detected WAR application via WEB-INF directory")
		return "Tomcat", nil
	}

	// Check for WAR files
	matches, err := filepath.Glob(filepath.Join(buildDir, "*.war"))
	if err == nil && len(matches) > 0 {
		t.context.Log.Debug("Detected WAR file: %s", matches[0])
		return "Tomcat", nil
	}

	return "", nil
}

// Supply installs Tomcat and dependencies
func (t *TomcatContainer) Supply() error {
	t.context.Log.BeginStep("Supplying Tomcat")

	// Determine Java version to select appropriate Tomcat version
	// Tomcat 10.x requires Java 11+, Tomcat 9.x supports Java 8-22
	javaHome := os.Getenv("JAVA_HOME")
	var dep libbuildpack.Dependency
	var err error

	if javaHome != "" {
		javaMajorVersion, versionErr := jres.DetermineJavaVersion(javaHome)
		if versionErr == nil {
			t.context.Log.Debug("Detected Java major version: %d", javaMajorVersion)

			// Select Tomcat version pattern based on Java version
			var versionPattern string
			if javaMajorVersion >= 11 {
				// Java 11+: Use Tomcat 10.x (Jakarta EE 9+)
				versionPattern = "10.x"
				t.context.Log.Info("Using Tomcat 10.x for Java %d", javaMajorVersion)
			} else {
				// Java 8-10: Use Tomcat 9.x (Java EE 8)
				versionPattern = "9.x"
				t.context.Log.Info("Using Tomcat 9.x for Java %d", javaMajorVersion)
			}

			// Resolve the version pattern to actual version using libbuildpack
			allVersions := t.context.Manifest.AllDependencyVersions("tomcat")
			resolvedVersion, err := libbuildpack.FindMatchingVersion(versionPattern, allVersions)
			if err == nil {
				dep.Name = "tomcat"
				dep.Version = resolvedVersion
				t.context.Log.Debug("Resolved Tomcat version pattern '%s' to %s", versionPattern, resolvedVersion)
			} else {
				t.context.Log.Warning("Unable to resolve Tomcat version pattern '%s': %s", versionPattern, err.Error())
			}
		} else {
			t.context.Log.Warning("Unable to determine Java version: %s", versionErr.Error())
		}
	}

	// Fallback to default version if we couldn't determine Java version
	if dep.Version == "" {
		dep, err = t.context.Manifest.DefaultVersion("tomcat")
		if err != nil {
			t.context.Log.Warning("Unable to determine default Tomcat version")
			// Final fallback to a known version
			dep.Name = "tomcat"
			dep.Version = "9.0.98"
		}
	}

	// Install Tomcat
	tomcatDir := filepath.Join(t.context.Stager.DepDir(), "tomcat")
	if err := t.context.Installer.InstallDependency(dep, tomcatDir); err != nil {
		return fmt.Errorf("failed to install Tomcat: %w", err)
	}

	t.context.Log.Info("Installed Tomcat version %s", dep.Version)

	// Find the actual Tomcat home (handle nested directories from tar extraction)
	// Apache Tomcat tarballs extract to apache-tomcat-X.Y.Z/ subdirectory
	tomcatHome, err := t.findTomcatHome(tomcatDir)
	if err != nil {
		return fmt.Errorf("failed to find Tomcat home: %w", err)
	}
	t.context.Log.Debug("Found Tomcat home at: %s", tomcatHome)

	// Write profile.d script to set CATALINA_HOME and CATALINA_BASE at runtime
	// At runtime, CF makes dependencies available at $DEPS_DIR/<idx>/
	// We need to point to the actual nested directory (e.g., apache-tomcat-X.Y.Z/)
	depsIdx := t.context.Stager.DepsIdx()

	// Get relative path from tomcatDir to tomcatHome for runtime
	relPath, err := filepath.Rel(tomcatDir, tomcatHome)
	if err != nil || relPath == "." {
		relPath = ""
	}

	var tomcatPath string
	if relPath == "" {
		tomcatPath = fmt.Sprintf("$DEPS_DIR/%s/tomcat", depsIdx)
	} else {
		tomcatPath = fmt.Sprintf("$DEPS_DIR/%s/tomcat/%s", depsIdx, relPath)
	}

	envContent := fmt.Sprintf(`export CATALINA_HOME=%s
export CATALINA_BASE=%s
`, tomcatPath, tomcatPath)

	if err := t.context.Stager.WriteProfileD("tomcat.sh", envContent); err != nil {
		t.context.Log.Warning("Could not write tomcat.sh profile.d script: %s", err.Error())
	} else {
		t.context.Log.Debug("Created profile.d script: tomcat.sh")
	}

	// Install Tomcat support libraries
	if err := t.installTomcatSupport(); err != nil {
		t.context.Log.Warning("Could not install Tomcat support: %s", err.Error())
	}

	// JVMKill agent is installed and configured by JRE component

	return nil
}

// installTomcatSupport installs Tomcat support libraries
func (t *TomcatContainer) installTomcatSupport() error {
	dep, err := t.context.Manifest.DefaultVersion("tomcat-lifecycle-support")
	if err != nil {
		return err
	}

	supportDir := filepath.Join(t.context.Stager.DepDir(), "tomcat-lifecycle-support")
	if err := t.context.Installer.InstallDependency(dep, supportDir); err != nil {
		return fmt.Errorf("failed to install Tomcat support: %w", err)
	}

	t.context.Log.Info("Installed Tomcat Lifecycle Support version %s", dep.Version)
	return nil
}

// Finalize performs final Tomcat configuration
func (t *TomcatContainer) Finalize() error {
	t.context.Log.BeginStep("Finalizing Tomcat")

	buildDir := t.context.Stager.BuildDir()
	tomcatDir := filepath.Join(t.context.Stager.DepDir(), "tomcat")

	// Find Tomcat home (may be in subdirectory like apache-tomcat-X.Y.Z)
	tomcatHome, err := t.findTomcatHome(tomcatDir)
	if err != nil {
		return fmt.Errorf("failed to find Tomcat home during finalize: %w", err)
	}

	// Check if we have an exploded WAR (WEB-INF directory in BuildDir)
	webInf := filepath.Join(buildDir, "WEB-INF")
	if _, err := os.Stat(webInf); err == nil {
		// Configure Tomcat to serve the application from BuildDir
		// This follows the immutable BuildDir pattern: application stays where deployed
		t.context.Log.Info("Configuring Tomcat to serve exploded WAR from BuildDir")

		// Create a custom context.xml file that points to BuildDir
		// At runtime, $HOME will resolve to the application directory
		if err := t.configureContextDocBase(tomcatHome); err != nil {
			return fmt.Errorf("failed to configure Tomcat context: %w", err)
		}

		t.context.Log.Info("Tomcat configured to serve application from $HOME (BuildDir)")
	}

	// Configure Tomcat support JAR in common classpath
	if err := t.configureTomcatSupport(tomcatHome); err != nil {
		t.context.Log.Warning("Could not configure Tomcat support: %s", err.Error())
	}

	// JVMKill agent is configured by JRE component in JAVA_OPTS

	return nil
}

// configureContextDocBase creates a context configuration that points to BuildDir
func (t *TomcatContainer) configureContextDocBase(tomcatHome string) error {
	// Create conf/Catalina/localhost directory if it doesn't exist
	contextDir := filepath.Join(tomcatHome, "conf", "Catalina", "localhost")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}

	// Create ROOT.xml context file
	// This tells Tomcat to serve the ROOT webapp from BuildDir (the application directory)
	// Tomcat supports ${propertyName} syntax for system properties in context.xml
	contextFile := filepath.Join(contextDir, "ROOT.xml")
	contextXML := `<?xml version="1.0" encoding="UTF-8"?>
<Context docBase="${user.home}/app" reloadable="false">
    <!-- Application served from BuildDir (/home/vcap/app), not moved to DepDir -->
    <!-- At runtime: user.home system property = /home/vcap, so we use ${user.home}/app -->
</Context>
`

	if err := os.WriteFile(contextFile, []byte(contextXML), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	t.context.Log.Debug("Created Tomcat context configuration: %s", contextFile)
	return nil
}

// configureTomcatSupport adds Tomcat support JAR to common classpath
func (t *TomcatContainer) configureTomcatSupport(tomcatHome string) error {
	supportDir := filepath.Join(t.context.Stager.DepDir(), "tomcat-lifecycle-support")

	// Check if support was installed
	if _, err := os.Stat(supportDir); os.IsNotExist(err) {
		return nil // Support not installed, skip
	}

	// Find the support JAR
	matches, err := filepath.Glob(filepath.Join(supportDir, "*.jar"))
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("tomcat support JAR not found in %s", supportDir)
	}

	supportJar := matches[0]

	// Create setenv.sh to add support JAR to classpath
	// This follows Tomcat's standard configuration mechanism
	binDir := filepath.Join(tomcatHome, "bin")
	setenvFile := filepath.Join(binDir, "setenv.sh")

	// Calculate runtime path to support JAR (relative to CATALINA_BASE)
	// At runtime: $CATALINA_BASE = /home/vcap/deps/0/tomcat/...
	// Support JAR is at: /home/vcap/deps/0/tomcat-lifecycle-support/...
	relPath, err := filepath.Rel(tomcatHome, supportJar)
	if err != nil {
		// If we can't calculate relative path, use absolute reference
		relPath = fmt.Sprintf("$CATALINA_BASE/../tomcat-lifecycle-support/%s", filepath.Base(supportJar))
	} else {
		relPath = fmt.Sprintf("$CATALINA_BASE/%s", relPath)
	}

	setenvContent := fmt.Sprintf(`#!/bin/bash
# Add Tomcat Lifecycle Support to classpath
export CLASSPATH="%s:$CLASSPATH"
`, relPath)

	if err := os.WriteFile(setenvFile, []byte(setenvContent), 0755); err != nil {
		return fmt.Errorf("failed to write setenv.sh: %w", err)
	}

	t.context.Log.Debug("Configured Tomcat support JAR in setenv.sh")
	return nil
}

// findTomcatHome finds the actual Tomcat home directory
// Apache Tomcat tarballs extract to apache-tomcat-X.Y.Z/ subdirectories
func (t *TomcatContainer) findTomcatHome(tomcatDir string) (string, error) {
	entries, err := os.ReadDir(tomcatDir)
	if err != nil {
		return "", fmt.Errorf("failed to read Tomcat directory: %w", err)
	}

	// Look for apache-tomcat-* subdirectory
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Check for apache-tomcat-* directory pattern
			if len(name) > 13 && name[:13] == "apache-tomcat" {
				path := filepath.Join(tomcatDir, name)
				// Verify it has bin/catalina.sh
				if _, err := os.Stat(filepath.Join(path, "bin", "catalina.sh")); err == nil {
					return path, nil
				}
			}
		}
	}

	// If no subdirectory found, check if tomcatDir itself is valid
	if _, err := os.Stat(filepath.Join(tomcatDir, "bin", "catalina.sh")); err == nil {
		return tomcatDir, nil
	}

	return "", fmt.Errorf("could not find valid Tomcat home in %s", tomcatDir)
}

// Release returns the Tomcat startup command
// Uses $CATALINA_HOME which is set by profile.d/tomcat.sh at runtime
func (t *TomcatContainer) Release() (string, error) {
	// Use $CATALINA_HOME environment variable set by profile.d script
	// Profile.d scripts run BEFORE the release command at runtime (same as $JAVA_HOME)
	cmd := "$CATALINA_HOME/bin/catalina.sh run"

	return cmd, nil
}
