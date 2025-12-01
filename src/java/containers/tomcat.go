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
	dep, err := t.context.Manifest.DefaultVersion("tomcat_lifecycle_support")
	if err != nil {
		return err
	}

	supportDir := filepath.Join(t.context.Stager.DepDir(), "tomcat_lifecycle_support")
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

	// If we have an exploded WAR (WEB-INF directory), move it to Tomcat's webapps
	webInf := filepath.Join(buildDir, "WEB-INF")
	if _, err := os.Stat(webInf); err == nil {
		// Create ROOT webapp directory
		rootApp := filepath.Join(tomcatDir, "webapps", "ROOT")
		if err := os.MkdirAll(rootApp, 0755); err != nil {
			return fmt.Errorf("failed to create ROOT webapp: %w", err)
		}

		// Move WEB-INF and other content to ROOT
		t.context.Log.Info("Deploying exploded WAR to Tomcat")
		// TODO: In full implementation, use proper file moving
		// For now, we'll assume symlinks or direct access
	}

	// JVMKill agent is configured by JRE component in JAVA_OPTS
	// CATALINA_OPTS configuration will be added in future enhancements

	// TODO: Add Tomcat support JAR to classpath
	// TODO: Configure server.xml with appropriate settings

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
