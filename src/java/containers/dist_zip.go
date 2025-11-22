package containers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DistZipContainer handles distribution ZIP applications
// (applications with bin/ and lib/ structure, typically from Gradle's distZip)
type DistZipContainer struct {
	context     *Context
	startScript string
}

// NewDistZipContainer creates a new Dist ZIP container
func NewDistZipContainer(ctx *Context) *DistZipContainer {
	return &DistZipContainer{
		context: ctx,
	}
}

// Detect checks if this is a Dist ZIP application
func (d *DistZipContainer) Detect() (string, error) {
	buildDir := d.context.Stager.BuildDir()

	// Check for bin/ and lib/ directories at root (typical distZip structure)
	binDir := filepath.Join(buildDir, "bin")
	libDir := filepath.Join(buildDir, "lib")

	binStat, binErr := os.Stat(binDir)
	libStat, libErr := os.Stat(libDir)

	if binErr == nil && libErr == nil && binStat.IsDir() && libStat.IsDir() {
		// Exclude Play Framework applications
		if d.isPlayFramework(libDir) {
			d.context.Log.Debug("Rejecting Dist ZIP detection - Play Framework JAR found")
			return "", nil
		}

		// Check for startup scripts in bin/
		entries, err := os.ReadDir(binDir)
		if err == nil && len(entries) > 0 {
			// Find a non-.bat script (Unix startup script)
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
					d.startScript = entry.Name()
					d.context.Log.Debug("Detected Dist ZIP application with start script: %s", d.startScript)
					return "Dist ZIP", nil
				}
			}
		}
	}

	// Check for bin/ and lib/ directories in application-root (alternative structure)
	binDirApp := filepath.Join(buildDir, "application-root", "bin")
	libDirApp := filepath.Join(buildDir, "application-root", "lib")

	binStatApp, binErrApp := os.Stat(binDirApp)
	libStatApp, libErrApp := os.Stat(libDirApp)

	if binErrApp == nil && libErrApp == nil && binStatApp.IsDir() && libStatApp.IsDir() {
		// Exclude Play Framework applications
		if d.isPlayFramework(libDirApp) {
			d.context.Log.Debug("Rejecting Dist ZIP detection - Play Framework JAR found in application-root")
			return "", nil
		}

		// Check for startup scripts in bin/
		entriesApp, errApp := os.ReadDir(binDirApp)
		if errApp == nil && len(entriesApp) > 0 {
			// Find a non-.bat script (Unix startup script)
			for _, entry := range entriesApp {
				if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
					d.startScript = filepath.Join("application-root", "bin", entry.Name())
					d.context.Log.Debug("Detected Dist ZIP application (application-root) with start script: %s", d.startScript)
					return "Dist ZIP", nil
				}
			}
		}
	}

	return "", nil
}

// isPlayFramework checks if a lib directory contains Play Framework JARs
func (d *DistZipContainer) isPlayFramework(libDir string) bool {
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return false
	}

	// Check for Play Framework JAR patterns:
	// - com.typesafe.play.play_*.jar (Play 2.2+)
	// - play.play_*.jar (Play 2.0)
	// - play_*.jar (Play 2.1)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, "com.typesafe.play.play_") ||
			strings.HasPrefix(name, "play.play_") ||
			(strings.HasPrefix(name, "play_") && strings.HasSuffix(name, ".jar")) {
			return true
		}
	}

	return false
}

// Supply installs Dist ZIP dependencies
func (d *DistZipContainer) Supply() error {
	d.context.Log.BeginStep("Supplying Dist ZIP")

	// For Dist ZIP apps, the structure is already provided
	// We may need to:
	// 1. Ensure scripts are executable
	// 2. Install support utilities

	// Make bin scripts executable
	if err := d.makeScriptsExecutable(); err != nil {
		d.context.Log.Warning("Could not make scripts executable: %s", err.Error())
	}

	// Install JVMKill agent
	if err := d.installJVMKillAgent(); err != nil {
		d.context.Log.Warning("Could not install JVMKill agent: %s", err.Error())
	}

	return nil
}

// makeScriptsExecutable ensures all scripts in bin/ are executable
func (d *DistZipContainer) makeScriptsExecutable() error {
	buildDir := d.context.Stager.BuildDir()

	// Try root bin/ directory
	binDir := filepath.Join(buildDir, "bin")
	entries, err := os.ReadDir(binDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
				scriptPath := filepath.Join(binDir, entry.Name())
				if err := os.Chmod(scriptPath, 0755); err != nil {
					d.context.Log.Warning("Could not make %s executable: %s", entry.Name(), err.Error())
				}
			}
		}
	}

	// Try application-root/bin/ directory
	binDirApp := filepath.Join(buildDir, "application-root", "bin")
	entriesApp, errApp := os.ReadDir(binDirApp)
	if errApp == nil {
		for _, entry := range entriesApp {
			if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
				scriptPath := filepath.Join(binDirApp, entry.Name())
				if err := os.Chmod(scriptPath, 0755); err != nil {
					d.context.Log.Warning("Could not make %s executable: %s", entry.Name(), err.Error())
				}
			}
		}
	}

	return nil
}

// installJVMKillAgent installs the JVMKill agent
func (d *DistZipContainer) installJVMKillAgent() error {
	dep, err := d.context.Manifest.DefaultVersion("jvmkill")
	if err != nil {
		return err
	}

	jvmkillPath := filepath.Join(d.context.Stager.DepDir(), "jvmkill")
	if err := d.context.Installer.InstallDependency(dep, jvmkillPath); err != nil {
		return fmt.Errorf("failed to install JVMKill: %w", err)
	}

	d.context.Log.Info("Installed JVMKill agent version %s", dep.Version)
	return nil
}

// Finalize performs final Dist ZIP configuration
func (d *DistZipContainer) Finalize() error {
	d.context.Log.BeginStep("Finalizing Dist ZIP")

	// Configure JAVA_OPTS to be picked up by startup scripts
	javaOpts := []string{
		"-Djava.io.tmpdir=$TMPDIR",
		"-XX:+ExitOnOutOfMemoryError",
	}

	// Add JVMKill agent if available
	jvmkillSO := filepath.Join(d.context.Stager.DepDir(), "jvmkill", "jvmkill.so")
	if _, err := os.Stat(jvmkillSO); err == nil {
		javaOpts = append(javaOpts, fmt.Sprintf("-agentpath:%s", jvmkillSO))
	}

	// Most distZip scripts respect JAVA_OPTS environment variable
	// Write JAVA_OPTS for the startup script to use
	if err := d.context.Stager.WriteEnvFile("JAVA_OPTS",
		strings.Join(javaOpts, " ")); err != nil {
		return fmt.Errorf("failed to write JAVA_OPTS: %w", err)
	}

	return nil
}

// Release returns the Dist ZIP startup command
func (d *DistZipContainer) Release() (string, error) {
	// Use the detected start script
	if d.startScript == "" {
		// Try to detect again
		if _, err := d.Detect(); err != nil || d.startScript == "" {
			return "", fmt.Errorf("no start script found in bin/ directory")
		}
	}

	// If the start script already contains a path (application-root case), use it as-is
	if strings.Contains(d.startScript, "/") {
		return d.startScript, nil
	}

	// Otherwise, prepend bin/ (root structure case)
	cmd := filepath.Join("bin", d.startScript)
	return cmd, nil
}
