package frameworks

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// JavaOptsFramework implements custom JAVA_OPTS configuration
type JavaOptsFramework struct {
	context *Context
}

// JavaOptsConfig represents the java_opts.yml configuration
type JavaOptsConfig struct {
	FromEnvironment bool     `yaml:"from_environment"`
	JavaOpts        []string `yaml:"java_opts"`
}

// NewJavaOptsFramework creates a new Java Opts framework instance
func NewJavaOptsFramework(ctx *Context) *JavaOptsFramework {
	return &JavaOptsFramework{context: ctx}
}

// Detect always returns true (universal framework for JAVA_OPTS configuration)
func (j *JavaOptsFramework) Detect() (string, error) {
	// Check if there's any configuration to apply
	config, err := j.loadConfig()
	if err != nil {
		j.context.Log.Debug("Failed to load java_opts config: %s", err.Error())
		return "", nil
	}

	// Detect if there are any custom java_opts or if from_environment is enabled
	if len(config.JavaOpts) > 0 || config.FromEnvironment {
		return "Java Opts", nil
	}

	return "", nil
}

// Supply does nothing (no dependencies to install)
func (j *JavaOptsFramework) Supply() error {
	// Java Opts framework only configures environment in finalize phase
	return nil
}

// Finalize applies the JAVA_OPTS configuration
func (j *JavaOptsFramework) Finalize() error {
	j.context.Log.BeginStep("Configuring Java Opts")

	// Load configuration
	config, err := j.loadConfig()
	if err != nil {
		j.context.Log.Warning("Failed to load java_opts config: %s", err.Error())
		return nil // Don't fail the build
	}

	var javaOpts []string

	// Add configured java_opts from config file
	if len(config.JavaOpts) > 0 {
		j.context.Log.Info("Adding configured JAVA_OPTS: %v", config.JavaOpts)
		javaOpts = append(javaOpts, config.JavaOpts...)
	}

	// Add $JAVA_OPTS from environment if from_environment is true
	if config.FromEnvironment {
		j.context.Log.Info("Including JAVA_OPTS from environment at runtime")
		// Add a placeholder that will be expanded at runtime in the startup script
		javaOpts = append(javaOpts, "${JAVA_OPTS}")
	}

	// If no opts to add, skip
	if len(javaOpts) == 0 {
		j.context.Log.Info("No JAVA_OPTS to configure")
		return nil
	}

	// Join all opts into a single string
	optsString := strings.Join(javaOpts, " ")

	// Append to existing JAVA_OPTS environment file (don't overwrite)
	if err := j.context.Stager.WriteEnvFile("JAVA_OPTS", optsString); err != nil {
		return fmt.Errorf("failed to set JAVA_OPTS: %w", err)
	}

	j.context.Log.Info("Configured JAVA_OPTS")
	return nil
}

// loadConfig loads the java_opts.yml configuration
func (j *JavaOptsFramework) loadConfig() (*JavaOptsConfig, error) {
	config := &JavaOptsConfig{
		FromEnvironment: true, // Default to true (matches config file)
		JavaOpts:        []string{},
	}

	// Check for JBP_CONFIG_JAVA_OPTS override
	configOverride := os.Getenv("JBP_CONFIG_JAVA_OPTS")
	if configOverride != "" {
		// Parse YAML from environment variable
		if err := yaml.Unmarshal([]byte(configOverride), config); err != nil {
			return nil, fmt.Errorf("failed to parse JBP_CONFIG_JAVA_OPTS: %w", err)
		}
		return config, nil
	}

	// Load from config file (java_opts.yml)
	configPath := j.context.Manifest.RootDir() + "/config/java_opts.yml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file not found is OK - use defaults
		return config, nil
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse java_opts.yml: %w", err)
	}

	return config, nil
}
