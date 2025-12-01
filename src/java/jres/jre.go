package jres

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

// JRE represents a Java Runtime Environment provider
type JRE interface {
	// Name returns the name of this JRE provider (e.g., "OpenJDK", "Zulu")
	Name() string

	// Detect returns true if this JRE should be used
	Detect() (bool, error)

	// Supply installs the JRE and its components (memory calculator, jvmkill)
	Supply() error

	// Finalize performs any final JRE configuration
	Finalize() error

	// JavaHome returns the path to JAVA_HOME
	JavaHome() string

	// Version returns the installed JRE version
	Version() string
}

// Context holds shared dependencies for JRE providers
type Context struct {
	Stager    *libbuildpack.Stager
	Manifest  *libbuildpack.Manifest
	Installer *libbuildpack.Installer
	Log       *libbuildpack.Logger
	Command   *libbuildpack.Command
}

// Registry manages multiple JRE providers
type Registry struct {
	ctx       *Context
	providers []JRE
}

// NewRegistry creates a new JRE registry
func NewRegistry(ctx *Context) *Registry {
	return &Registry{
		ctx:       ctx,
		providers: []JRE{},
	}
}

// Register adds a JRE provider to the registry
func (r *Registry) Register(jre JRE) {
	r.providers = append(r.providers, jre)
}

// Detect finds the first JRE provider that should be used
// Returns the JRE, its name, and any error
func (r *Registry) Detect() (JRE, string, error) {
	for _, jre := range r.providers {
		detected, err := jre.Detect()
		if err != nil {
			r.ctx.Log.Warning("Error detecting JRE %s: %s", jre.Name(), err.Error())
			continue
		}
		if detected {
			return jre, jre.Name(), nil
		}
	}
	return nil, "", nil
}

// Component represents a JRE component (memory calculator, jvmkill, etc.)
type Component interface {
	// Name returns the component name
	Name() string

	// Supply installs the component
	Supply() error

	// Finalize performs final configuration
	Finalize() error
}

// BaseComponent provides common functionality for JRE components
type BaseComponent struct {
	Ctx         *Context
	JREDir      string
	JREVersion  string
	ComponentID string
}

// Memory calculator constants
const (
	DefaultStackThreads = 250
	DefaultHeadroom     = 0
	Java9ClassCount     = 42215 // Classes in Java 9+ JRE
)

// Helper functions

// DetectJREByEnv checks environment variables for JRE selection
// Supports JBP_CONFIG_OPEN_JDK_JRE, etc.
func DetectJREByEnv(jreName string) bool {
	envKey := fmt.Sprintf("JBP_CONFIG_%s", strings.ToUpper(strings.ReplaceAll(jreName, "-", "_")))
	return os.Getenv(envKey) != ""
}

// GetJREVersion gets the desired JRE version from environment or uses default
// Supports BP_JAVA_VERSION (simple version) and JBP_CONFIG_<JRE_NAME> (complex config)
func GetJREVersion(ctx *Context, jreName string) (libbuildpack.Dependency, error) {
	// Check for simple BP_JAVA_VERSION environment variable first
	// Format: "8", "11", "17", "21", etc. or version patterns like "11.+", "17.*"
	if bpVersion := os.Getenv("BP_JAVA_VERSION"); bpVersion != "" {
		ctx.Log.Debug("Using Java version from BP_JAVA_VERSION: %s", bpVersion)

		// Normalize version to a pattern that FindMatchingVersion understands
		versionPattern := normalizeVersionPattern(bpVersion)

		// Get all available versions for this JRE
		availableVersions := ctx.Manifest.AllDependencyVersions(jreName)
		if len(availableVersions) == 0 {
			return libbuildpack.Dependency{}, fmt.Errorf("no versions found for %s", jreName)
		}

		// Find the highest matching version
		matchedVersion, err := libbuildpack.FindMatchingVersion(versionPattern, availableVersions)
		if err != nil {
			ctx.Log.Warning("Could not find %s matching version %s: %s", jreName, versionPattern, err.Error())
			return libbuildpack.Dependency{}, fmt.Errorf("no version of %s matching %s found", jreName, versionPattern)
		}

		ctx.Log.Debug("Resolved %s version %s from pattern %s", jreName, matchedVersion, versionPattern)
		return libbuildpack.Dependency{Name: jreName, Version: matchedVersion}, nil
	}

	// Check for legacy JBP_CONFIG_<JRE_NAME> environment variable
	envKey := fmt.Sprintf("JBP_CONFIG_%s", strings.ToUpper(strings.ReplaceAll(jreName, "-", "_")))
	if envVal := os.Getenv(envKey); envVal != "" {
		// Parse version from env (e.g., '{jre: {version: 11.+}}')
		// For now, simplified - just log it
		ctx.Log.Debug("JRE version override from %s: %s", envKey, envVal)
		// TODO: Parse YAML-like config from envVal
	}

	// Get default version from manifest (no version constraint)
	dep, err := ctx.Manifest.DefaultVersion(jreName)
	if err != nil {
		return libbuildpack.Dependency{}, err
	}

	return dep, nil
}

// normalizeVersionPattern converts user-friendly version strings to manifest patterns
// Examples: "8" -> "8.*", "11" -> "11.*", "17.0" -> "17.0.*", "11.+" -> "11.+"
func normalizeVersionPattern(version string) string {
	// If already has wildcard, return as-is
	if strings.Contains(version, "*") || strings.Contains(version, "+") {
		return version
	}

	// Otherwise append ".*" to match any patch version
	return version + ".*"
}

// SetupJavaHome sets JAVA_HOME and related environment variables
func SetupJavaHome(ctx *Context, javaHome string) error {
	// Find actual JRE directory (usually jdk-* or jre-* subdirectory)
	entries, err := os.ReadDir(javaHome)
	if err != nil {
		return fmt.Errorf("failed to read JRE directory: %w", err)
	}

	// Look for jdk-* or jre-* subdirectory
	var actualJavaHome string
	var subdirectory string
	for _, entry := range entries {
		if entry.IsDir() && (strings.HasPrefix(entry.Name(), "jdk") || strings.HasPrefix(entry.Name(), "jre")) {
			actualJavaHome = filepath.Join(javaHome, entry.Name())
			subdirectory = entry.Name()
			break
		}
	}

	// If no subdirectory found, use the javaHome directly
	if actualJavaHome == "" {
		actualJavaHome = javaHome
		subdirectory = ""
	}

	// Construct runtime path for profile.d script using $DEPS_DIR
	// At runtime, CF sets $DEPS_DIR (e.g., /home/vcap/deps) and makes dependencies available at $DEPS_DIR/<idx>/
	// The javaHome parameter is something like /tmp/app/.cloudfoundry/0/jre (staging time)
	// We need to construct $DEPS_DIR/0/jre/<subdirectory> (runtime)
	depsIdx := ctx.Stager.DepsIdx()
	var runtimeJavaHome string
	if subdirectory != "" {
		runtimeJavaHome = fmt.Sprintf("$DEPS_DIR/%s/jre/%s", depsIdx, subdirectory)
	} else {
		runtimeJavaHome = fmt.Sprintf("$DEPS_DIR/%s/jre", depsIdx)
	}

	// Write environment variables to profile.d using runtime $HOME path
	envContent := fmt.Sprintf(`export JAVA_HOME=%s
export JRE_HOME=%s
export PATH=$JAVA_HOME/bin:$PATH
`, runtimeJavaHome, runtimeJavaHome)

	if err := ctx.Stager.WriteProfileD("java.sh", envContent); err != nil {
		return fmt.Errorf("failed to write java.sh: %w", err)
	}

	// Also set for current process (use absolute path for staging time)
	os.Setenv("JAVA_HOME", actualJavaHome)
	os.Setenv("JRE_HOME", actualJavaHome)
	os.Setenv("PATH", filepath.Join(actualJavaHome, "bin")+":"+os.Getenv("PATH"))

	ctx.Log.Info("Set JAVA_HOME to %s (runtime: %s)", actualJavaHome, runtimeJavaHome)

	return nil
}

// DetermineJavaVersion determines the major Java version from the installed JRE
func DetermineJavaVersion(javaHome string) (int, error) {
	// Try to read release file
	releaseFile := filepath.Join(javaHome, "release")
	if data, err := os.ReadFile(releaseFile); err == nil {
		// Parse JAVA_VERSION="1.8.0_422" or JAVA_VERSION="17.0.13"
		content := string(data)
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "JAVA_VERSION=") {
				version := strings.Trim(strings.TrimPrefix(line, "JAVA_VERSION="), "\"")
				// Parse major version
				if strings.HasPrefix(version, "1.8") {
					return 8, nil
				}
				// For Java 9+, major version is the first number
				parts := strings.Split(version, ".")
				if len(parts) > 0 {
					var major int
					fmt.Sscanf(parts[0], "%d", &major)
					return major, nil
				}
			}
		}
	}

	// Default to 17 if we can't determine
	return 17, nil
}

// WriteJavaOpts writes JAVA_OPTS to a profile.d script for runtime export
func WriteJavaOpts(ctx *Context, opts string) error {
	profileDir := filepath.Join(ctx.Stager.BuildDir(), ".profile.d")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create .profile.d directory: %w", err)
	}

	profileScript := filepath.Join(profileDir, "java_opts.sh")

	// Append to existing JAVA_OPTS if file exists
	var scriptContent string
	if existing, err := os.ReadFile(profileScript); err == nil {
		// File exists - extract current JAVA_OPTS value and append
		scriptContent = string(existing)
		// Remove the trailing newline if present
		scriptContent = strings.TrimSuffix(scriptContent, "\n")
		// Append new opts to the export line
		scriptContent = strings.Replace(scriptContent, "${JAVA_OPTS:-}", "${JAVA_OPTS:-} "+opts, 1)
		scriptContent += "\n"
	} else {
		// Create new profile.d script with export statement
		scriptContent = fmt.Sprintf("export JAVA_OPTS=\"${JAVA_OPTS:-%s}\"\n", opts)
	}

	if err := os.WriteFile(profileScript, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write profile.d/java_opts.sh: %w", err)
	}

	return nil
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
