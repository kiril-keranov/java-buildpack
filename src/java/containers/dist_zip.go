package containers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// Finalize performs final Dist ZIP configuration
func (d *DistZipContainer) Finalize() error {
	d.context.Log.BeginStep("Finalizing Dist ZIP")
	d.context.Log.Info("DistZip Finalize: Starting (startScript=%s)", d.startScript)

	// Write profile.d script to set DIST_ZIP_HOME and PATH at runtime
	// At runtime, CF makes the application available at $HOME
	// This ensures the startup script directory is in PATH

	// Determine the script directory based on start script location
	var scriptDir string
	if strings.Contains(d.startScript, "/") {
		// application-root case: extract directory from script path
		scriptDir = filepath.Dir(d.startScript)
	} else {
		// root structure case: script in bin/
		scriptDir = "bin"
	}

	envContent := fmt.Sprintf(`export DEPS_DIR=${DEPS_DIR:-/home/vcap/deps}
export DIST_ZIP_HOME=$HOME
export DIST_ZIP_BIN=$HOME/%s
export PATH=$DIST_ZIP_BIN:$PATH
`, scriptDir)

	if err := d.context.Stager.WriteProfileD("dist_zip.sh", envContent); err != nil {
		d.context.Log.Warning("Could not write dist_zip.sh profile.d script: %s", err.Error())
	} else {
		d.context.Log.Debug("Created profile.d script: dist_zip.sh")
	}

	// Augment startup script CLASSPATH with additional libraries
	// This matches Ruby buildpack behavior: modify script directly instead of using .profile.d
	if err := d.augmentStartupScript(); err != nil {
		d.context.Log.Warning("Could not augment startup script: %s", err.Error())
	} else {
		d.context.Log.Info("DistZip Finalize: Successfully augmented startup script")
	}

	// Configure JAVA_OPTS to be picked up by startup scripts
	// Note: JVMKill agent is configured by the JRE component via .profile.d/java_opts.sh
	javaOpts := []string{
		"-Djava.io.tmpdir=$TMPDIR",
		"-XX:+ExitOnOutOfMemoryError",
	}

	// Most distZip scripts respect JAVA_OPTS environment variable
	// Write JAVA_OPTS for the startup script to use
	if err := d.context.Stager.WriteEnvFile("JAVA_OPTS",
		strings.Join(javaOpts, " ")); err != nil {
		return fmt.Errorf("failed to write JAVA_OPTS: %w", err)
	}

	return nil
}

// augmentStartupScript modifies the startup script to prepend additional libraries to CLASSPATH
// This follows Ruby buildpack's approach from lib/java_buildpack/container/dist_zip_like.rb
func (d *DistZipContainer) augmentStartupScript() error {
	buildDir := d.context.Stager.BuildDir()

	// Determine startup script path
	var scriptPath string
	if strings.Contains(d.startScript, "/") {
		// application-root case: script path includes directory
		scriptPath = filepath.Join(buildDir, d.startScript)
	} else {
		// root structure case: script in bin/
		scriptPath = filepath.Join(buildDir, "bin", d.startScript)
	}

	d.context.Log.Info("DistZip augmentStartupScript: scriptPath=%s, buildDir=%s", scriptPath, buildDir)

	// Read startup script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		d.context.Log.Error("DistZip augmentStartupScript: Failed to read script at %s: %s", scriptPath, err.Error())
		return fmt.Errorf("failed to read startup script: %w", err)
	}

	d.context.Log.Info("DistZip augmentStartupScript: Read %d bytes from startup script", len(content))
	scriptContent := string(content)

	// Collect additional libraries (JVMKill agent, frameworks, etc.)
	additionalLibs := d.collectAdditionalLibraries()
	d.context.Log.Info("DistZip augmentStartupScript: Found %d additional libraries: %v", len(additionalLibs), additionalLibs)
	if len(additionalLibs) == 0 {
		d.context.Log.Info("DistZip augmentStartupScript: No additional libraries to add to CLASSPATH, skipping augmentation")
		return nil
	}

	// Try augmenting CLASSPATH using two patterns (matching Ruby buildpack):
	// 1. declare -r app_classpath="..." (newer Gradle format)
	// 2. CLASSPATH=... (older format)

	modified := false

	// Pattern 1: declare -r app_classpath="..."
	// Example: declare -r app_classpath="$app_home/lib/myapp.jar:$app_home/lib/dep.jar"
	// We prepend: declare -r app_classpath="$app_home/.deps/0/jvmkill/jvmkill.so:$app_home/lib/myapp.jar:..."
	appClasspathPattern := regexp.MustCompile(`(?m)^declare -r app_classpath="(.+)"$`)
	if appClasspathPattern.MatchString(scriptContent) {
		d.context.Log.Info("DistZip augmentStartupScript: Found app_classpath pattern in startup script")

		// Build classpath prefix using $app_home (relative to script location)
		classpathPrefix := d.buildClasspathPrefix(additionalLibs, "$app_home")
		d.context.Log.Info("DistZip augmentStartupScript: Built classpath prefix: %s", classpathPrefix)

		scriptContent = appClasspathPattern.ReplaceAllString(scriptContent,
			fmt.Sprintf(`declare -r app_classpath="%s:$1"`, classpathPrefix))
		modified = true
	}

	// Pattern 2: CLASSPATH=...
	// Example: CLASSPATH=$APP_HOME/lib/myapp.jar:$APP_HOME/lib/dep.jar
	// We prepend: CLASSPATH=$APP_HOME/.deps/0/jvmkill/jvmkill.so:$APP_HOME/lib/myapp.jar:...
	if !modified {
		classpathPattern := regexp.MustCompile(`(?m)^CLASSPATH=(.+)$`)
		if classpathPattern.MatchString(scriptContent) {
			d.context.Log.Info("DistZip augmentStartupScript: Found CLASSPATH pattern in startup script")

			// Build classpath prefix using $APP_HOME (absolute path from root)
			classpathPrefix := d.buildClasspathPrefix(additionalLibs, "$APP_HOME")
			d.context.Log.Info("DistZip augmentStartupScript: Built classpath prefix: %s", classpathPrefix)

			// Use ReplaceAllStringFunc to avoid $ being interpreted as regex backreference
			scriptContent = classpathPattern.ReplaceAllStringFunc(scriptContent, func(match string) string {
				// Extract original CLASSPATH value (everything after "CLASSPATH=")
				originalClasspath := strings.TrimPrefix(match, "CLASSPATH=")
				replacement := fmt.Sprintf("CLASSPATH=%s:%s", classpathPrefix, originalClasspath)
				d.context.Log.Info("DistZip augmentStartupScript: Matched line: %s", match)
				d.context.Log.Info("DistZip augmentStartupScript: Original classpath: %s", originalClasspath)
				d.context.Log.Info("DistZip augmentStartupScript: Replacement line: %s", replacement)
				return replacement
			})
			modified = true
			d.context.Log.Info("DistZip augmentStartupScript: Script modification complete")
		}
	}

	if !modified {
		d.context.Log.Warning("DistZip augmentStartupScript: No CLASSPATH pattern found in startup script - cannot augment")
		d.context.Log.Info("DistZip augmentStartupScript: First 500 chars of script:\n%s", scriptContent[:min(500, len(scriptContent))])
		return nil
	}

	// Write modified script back
	d.context.Log.Info("DistZip augmentStartupScript: About to write modified script (first 1000 chars):\n%s", scriptContent[:min(1000, len(scriptContent))])
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		d.context.Log.Error("DistZip augmentStartupScript: Failed to write modified script: %s", err.Error())
		return fmt.Errorf("failed to write modified startup script: %w", err)
	}

	d.context.Log.Info("DistZip augmentStartupScript: Successfully wrote modified script to %s", scriptPath)
	d.context.Log.Info("Augmented startup script CLASSPATH with %d additional libraries", len(additionalLibs))
	return nil
}

// collectAdditionalLibraries gathers all additional libraries that should be added to CLASSPATH
// This includes framework-provided JAR libraries installed during supply phase
func (d *DistZipContainer) collectAdditionalLibraries() []string {
	var libs []string
	depsDir := d.context.Stager.DepDir()

	// Scan $DEPS_DIR/0/ for all framework directories
	entries, err := os.ReadDir(depsDir)
	if err != nil {
		d.context.Log.Debug("Unable to read deps directory: %s", err.Error())
		return libs
	}

	// Iterate through each framework directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		frameworkDir := filepath.Join(depsDir, entry.Name())

		// Find all *.jar files in this framework directory
		jarPattern := filepath.Join(frameworkDir, "*.jar")
		matches, err := filepath.Glob(jarPattern)
		if err != nil {
			d.context.Log.Debug("Error globbing JARs in %s: %s", frameworkDir, err.Error())
			continue
		}

		// Add all found JARs to the list
		// NOTE: Native libraries (.so, .dylib files like jvmkill) are NOT added here
		// Native libraries are loaded via -agentpath in JAVA_OPTS
		for _, jar := range matches {
			// Skip native libraries - only include .jar files
			if filepath.Ext(jar) == ".jar" {
				libs = append(libs, jar)
			}
		}
	}

	return libs
}

// buildClasspathPrefix constructs the CLASSPATH prefix from additional libraries
// baseVar is either "$app_home" (relative to script location) or "$APP_HOME" (absolute from root)
func (d *DistZipContainer) buildClasspathPrefix(libs []string, baseVar string) string {
	depsDir := d.context.Stager.DepDir()
	buildDir := d.context.Stager.BuildDir()

	var classpathParts []string
	for _, lib := range libs {
		var runtimePath string

		// Check if library is in deps directory (e.g., JVMKill agent)
		if strings.HasPrefix(lib, depsDir) {
			// Convert staging absolute path to runtime-relative path
			// Staging: /tmp/staging/deps/0/jre/bin/jvmkill-1.16.0.so
			// Runtime: $APP_HOME/../deps/0/jre/bin/jvmkill-1.16.0.so (relative to app directory)
			// Note: We use baseVar/../deps instead of $DEPS_DIR because the startup script
			// runs before .profile.d scripts are sourced, so $DEPS_DIR is not yet available.
			// At runtime: $HOME=/home/vcap/app, and deps is at /home/vcap/deps (sibling directory)
			relPath := strings.TrimPrefix(lib, depsDir)
			relPath = strings.TrimPrefix(relPath, "/") // Remove leading slash
			relPath = filepath.ToSlash(relPath)        // Normalize slashes
			runtimePath = fmt.Sprintf("%s/../deps/0/%s", baseVar, relPath)
		} else if strings.HasPrefix(lib, buildDir) {
			// Library is in build directory, calculate relative path from app root
			relPath, err := filepath.Rel(buildDir, lib)
			if err != nil {
				d.context.Log.Warning("Could not calculate relative path for %s: %s", lib, err.Error())
				continue
			}
			relPath = filepath.ToSlash(relPath)
			runtimePath = fmt.Sprintf("%s/%s", baseVar, relPath)
		} else {
			// Fallback: library path doesn't match expected patterns
			d.context.Log.Warning("Library path %s doesn't match deps or build directory, using as-is", lib)
			runtimePath = lib
		}

		classpathParts = append(classpathParts, runtimePath)
	}

	return strings.Join(classpathParts, ":")
}

// Release returns the Dist ZIP startup command
// Uses absolute path to ensure script is found at runtime
func (d *DistZipContainer) Release() (string, error) {
	// Use the detected start script
	if d.startScript == "" {
		// Try to detect again
		if _, err := d.Detect(); err != nil || d.startScript == "" {
			return "", fmt.Errorf("no start script found in bin/ directory")
		}
	}

	// Determine the script directory based on start script location
	var scriptDir string
	if strings.Contains(d.startScript, "/") {
		// application-root case: extract directory from script path
		scriptDir = filepath.Dir(d.startScript)
	} else {
		// root structure case: script in bin/
		scriptDir = "bin"
	}

	// Extract just the script name (remove any directory path)
	scriptName := filepath.Base(d.startScript)

	// Use absolute path $HOME/<scriptDir>/<scriptName>
	// This eliminates dependency on profile.d script execution order
	// At runtime, CF makes the application available at $HOME
	cmd := fmt.Sprintf("$HOME/%s/%s", scriptDir, scriptName)

	return cmd, nil
}
