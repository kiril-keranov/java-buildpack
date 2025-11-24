package frameworks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

// SpringAutoReconfigurationFramework implements Spring Auto-reconfiguration support for Cloud Foundry
// This framework automatically reconfigures Spring applications to use Cloud Foundry services
// Note: This is deprecated in favor of java-cfenv, but still widely used
type SpringAutoReconfigurationFramework struct {
	context *Context
}

// NewSpringAutoReconfigurationFramework creates a new Spring Auto-reconfiguration framework instance
func NewSpringAutoReconfigurationFramework(ctx *Context) *SpringAutoReconfigurationFramework {
	return &SpringAutoReconfigurationFramework{context: ctx}
}

// Detect checks if Spring Auto-reconfiguration should be included
func (s *SpringAutoReconfigurationFramework) Detect() (string, error) {
	// Check if enabled in configuration
	enabled := s.isEnabled()
	if !enabled {
		return "", nil
	}

	// Check if Spring is present
	if !s.hasSpring() {
		return "", nil
	}

	// Don't enable if java-cfenv is already present
	if s.hasJavaCfEnv() {
		s.context.Log.Debug("java-cfenv detected, skipping Spring Auto-reconfiguration")
		return "", nil
	}

	return "Spring Auto-reconfiguration", nil
}

// Supply installs the Spring Auto-reconfiguration JAR
func (s *SpringAutoReconfigurationFramework) Supply() error {
	s.context.Log.BeginStep("Installing Spring Auto-reconfiguration")

	// Log deprecation warnings
	if s.hasSpringCloudConnectors() {
		s.context.Log.Warning("ATTENTION: The Spring Cloud Connectors library is present in your application. This library " +
			"has been in maintenance mode since July 2019 and will stop receiving all updates after Mar 2024.")
		s.context.Log.Warning("Please migrate to java-cfenv immediately. See https://via.vmw.com/EiBW for migration instructions.")
	}

	// Check again if java-cfenv framework is being installed
	if s.hasJavaCfEnv() {
		s.context.Log.Debug("java-cfenv present, skipping Spring Auto-reconfiguration installation")
		return nil
	}

	// Get Spring Auto-reconfiguration dependency from manifest
	dep, err := s.context.Manifest.DefaultVersion("auto-reconfiguration")
	if err != nil {
		s.context.Log.Warning("Unable to determine Spring Auto-reconfiguration version, using default")
		dep = libbuildpack.Dependency{
			Name:    "auto-reconfiguration",
			Version: "2.13.0", // Fallback version
		}
	}

	// Install Spring Auto-reconfiguration JAR
	autoReconfDir := filepath.Join(s.context.Stager.DepDir(), "spring_auto_reconfiguration")
	if err := s.context.Installer.InstallDependency(dep, autoReconfDir); err != nil {
		return fmt.Errorf("failed to install Spring Auto-reconfiguration: %w", err)
	}

	// The JAR will be added to classpath in finalize phase
	s.context.Log.Warning("ATTENTION: The Spring Auto Reconfiguration and shaded Spring Cloud Connectors libraries are " +
		"being installed. These projects have been deprecated, are no longer receiving updates and should " +
		"not be used going forward.")
	s.context.Log.Warning("If you are not using these libraries, set `JBP_CONFIG_SPRING_AUTO_RECONFIGURATION='{enabled: false}'` " +
		"to disable their installation and clear this warning message. The buildpack will switch its default " +
		"to disable by default after March 2023. Spring Auto Reconfiguration and its shaded Spring Cloud " +
		"Connectors will be removed from the buildpack after March 2024.")
	s.context.Log.Warning("If you are using these libraries, please migrate to java-cfenv immediately. " +
		"See https://via.vmw.com/EiBW for migration instructions. Once you upgrade this message will go away.")

	s.context.Log.Info("Installed Spring Auto-reconfiguration version %s", dep.Version)
	return nil
}

// Finalize performs final Spring Auto-reconfiguration configuration
func (s *SpringAutoReconfigurationFramework) Finalize() error {
	// Add the JAR to additional libraries (classpath)
	autoReconfDir := filepath.Join(s.context.Stager.DepDir(), "spring_auto_reconfiguration")
	jarPattern := filepath.Join(autoReconfDir, "auto-reconfiguration-*.jar")

	matches, err := filepath.Glob(jarPattern)
	if err != nil || len(matches) == 0 {
		// JAR not found, might not have been installed
		return nil
	}

	// Add to classpath via CLASSPATH environment variable
	classpath := os.Getenv("CLASSPATH")
	if classpath != "" {
		classpath += ":"
	}
	classpath += matches[0]

	if err := s.context.Stager.WriteEnvFile("CLASSPATH", classpath); err != nil {
		return fmt.Errorf("failed to set CLASSPATH for Spring Auto-reconfiguration: %w", err)
	}

	return nil
}

// isEnabled checks if Spring Auto-reconfiguration is enabled in configuration
func (s *SpringAutoReconfigurationFramework) isEnabled() bool {
	// Check JBP_CONFIG_SPRING_AUTO_RECONFIGURATION environment variable
	config := os.Getenv("JBP_CONFIG_SPRING_AUTO_RECONFIGURATION")
	if config != "" {
		// If explicitly configured, respect that setting
		// For now, we'll assume if it's set, it's to disable
		// A more robust implementation would parse the YAML/JSON
		return false
	}

	// Default to enabled (for now, will be disabled by default after March 2023)
	return true
}

// hasSpring checks if Spring Core is present in the application
func (s *SpringAutoReconfigurationFramework) hasSpring() bool {
	// Look for spring-core*.jar in the application
	pattern := filepath.Join(s.context.Stager.BuildDir(), "**", "spring-core*.jar")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}

	if len(matches) > 0 {
		return true
	}

	// Also check common locations
	commonPaths := []string{
		filepath.Join(s.context.Stager.BuildDir(), "WEB-INF", "lib", "spring-core*.jar"),
		filepath.Join(s.context.Stager.BuildDir(), "lib", "spring-core*.jar"),
		filepath.Join(s.context.Stager.BuildDir(), "BOOT-INF", "lib", "spring-core*.jar"),
	}

	for _, path := range commonPaths {
		matches, _ := filepath.Glob(path)
		if len(matches) > 0 {
			return true
		}
	}

	return false
}

// hasJavaCfEnv checks if java-cfenv is present in the application
func (s *SpringAutoReconfigurationFramework) hasJavaCfEnv() bool {
	// Check common locations for java-cfenv*.jar
	commonPaths := []string{
		filepath.Join(s.context.Stager.BuildDir(), "WEB-INF", "lib", "java-cfenv*.jar"),
		filepath.Join(s.context.Stager.BuildDir(), "lib", "java-cfenv*.jar"),
		filepath.Join(s.context.Stager.BuildDir(), "BOOT-INF", "lib", "java-cfenv*.jar"),
	}

	for _, path := range commonPaths {
		matches, _ := filepath.Glob(path)
		if len(matches) > 0 {
			return true
		}
	}

	// Also check if java_cf_env framework is being installed
	javaCfEnvDir := filepath.Join(s.context.Stager.DepDir(), "java_cf_env")
	if _, err := os.Stat(javaCfEnvDir); err == nil {
		return true
	}

	return false
}

// hasSpringCloudConnectors checks if Spring Cloud Connectors are present
func (s *SpringAutoReconfigurationFramework) hasSpringCloudConnectors() bool {
	patterns := []string{
		filepath.Join(s.context.Stager.BuildDir(), "**", "spring-cloud-cloudfoundry-connector*.jar"),
		filepath.Join(s.context.Stager.BuildDir(), "**", "spring-cloud-spring-service-connector*.jar"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}
