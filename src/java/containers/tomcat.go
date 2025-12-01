package containers

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Install Tomcat
	dep, err := t.context.Manifest.DefaultVersion("tomcat")
	if err != nil {
		t.context.Log.Warning("Unable to determine default Tomcat version")
		// Fallback to a known version
		dep.Name = "tomcat"
		dep.Version = "9.0.98"
	}

	tomcatDir := filepath.Join(t.context.Stager.DepDir(), "tomcat")
	if err := t.context.Installer.InstallDependency(dep, tomcatDir); err != nil {
		return fmt.Errorf("failed to install Tomcat: %w", err)
	}

	t.context.Log.Info("Installed Tomcat version %s", dep.Version)

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

	// CATALINA_OPTS configuration will be added in future enhancements

	// TODO: Add Tomcat support JAR to classpath
	// TODO: Configure server.xml with appropriate settings

	return nil
}

// Release returns the Tomcat startup command
func (t *TomcatContainer) Release() (string, error) {
	tomcatDir := filepath.Join(t.context.Stager.DepDir(), "tomcat")
	catalinaHome := tomcatDir
	catalinaBase := tomcatDir

	cmd := fmt.Sprintf("CATALINA_HOME=%s CATALINA_BASE=%s %s/bin/catalina.sh run",
		catalinaHome, catalinaBase, tomcatDir)

	return cmd, nil
}
