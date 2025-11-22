package containers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PlayContainer represents a Play Framework application container
type PlayContainer struct {
	context     *Context
	playType    string // "pre22_dist", "pre22_staged", "post22_dist", "post22_staged"
	playVersion string
	startScript string
	libDir      string
}

// NewPlayContainer creates a new Play Framework container
func NewPlayContainer(ctx *Context) *PlayContainer {
	return &PlayContainer{
		context: ctx,
	}
}

// Detect checks if this is a Play Framework application
func (p *PlayContainer) Detect() (string, error) {
	buildDir := p.context.Stager.BuildDir()

	p.context.Log.Debug("Play: Checking buildDir: %s", buildDir)

	// First, validate that we don't have ambiguous configuration (hybrid apps)
	if err := p.Validate(); err != nil {
		p.context.Log.Debug("Play: Validation failed: %v", err)
		return "", err
	}

	// Try to detect Play Framework type in order of specificity
	// Order matters to avoid ambiguous detection

	// 1. Try Post22Dist (Play 2.2+ distributed app in application-root/bin)
	p.context.Log.Debug("Play: Trying Post22Dist detection")
	if p.detectPost22Dist(buildDir) {
		p.context.Log.Info("Play: Detected Post22Dist - version %s", p.playVersion)
		return fmt.Sprintf("Play Framework %s", p.playVersion), nil
	}

	// 2. Try Post22Staged (Play 2.2+ staged app in bin/)
	p.context.Log.Debug("Play: Trying Post22Staged detection")
	if p.detectPost22Staged(buildDir) {
		p.context.Log.Info("Play: Detected Post22Staged - version %s", p.playVersion)
		return fmt.Sprintf("Play Framework %s", p.playVersion), nil
	}

	// 3. Try Pre22Dist (Play 2.0-2.1 distributed app in application-root/)
	p.context.Log.Debug("Play: Trying Pre22Dist detection")
	if p.detectPre22Dist(buildDir) {
		p.context.Log.Info("Play: Detected Pre22Dist - version %s", p.playVersion)
		return fmt.Sprintf("Play Framework %s", p.playVersion), nil
	}

	// 4. Try Pre22Staged (Play 2.0-2.1 staged app with start script at root)
	p.context.Log.Debug("Play: Trying Pre22Staged detection")
	if p.detectPre22Staged(buildDir) {
		p.context.Log.Info("Play: Detected Pre22Staged - version %s", p.playVersion)
		return fmt.Sprintf("Play Framework %s", p.playVersion), nil
	}

	p.context.Log.Debug("Play: No Play Framework detected")
	return "", nil
}

// detectPost22Dist detects Play 2.2+ distributed applications
// Structure: application-root/bin/<script>, application-root/lib/com.typesafe.play.play_*.jar
func (p *PlayContainer) detectPost22Dist(buildDir string) bool {
	// Check for application-root/bin/ directory
	binDir := filepath.Join(buildDir, "application-root", "bin")
	binStat, binErr := os.Stat(binDir)
	if binErr != nil || !binStat.IsDir() {
		return false
	}

	// Check for application-root/lib/ directory
	libDir := filepath.Join(buildDir, "application-root", "lib")
	libStat, libErr := os.Stat(libDir)
	if libErr != nil || !libStat.IsDir() {
		return false
	}

	// Find Play JAR in lib/ (com.typesafe.play.play_*.jar)
	playJar, version := p.findPlayJar(libDir)
	if playJar == "" {
		return false
	}

	// Parse version - must be 2.2 or higher
	if !p.isPost22Version(version) {
		return false
	}

	// Find start script in bin/ (non-.bat file)
	startScript := p.findStartScript(binDir)
	if startScript == "" {
		return false
	}

	p.playType = "post22_dist"
	p.playVersion = version
	p.startScript = filepath.Join("application-root", "bin", startScript)
	p.libDir = libDir
	p.context.Log.Debug("Detected Play Framework %s (Post22Dist)", version)
	return true
}

// detectPost22Staged detects Play 2.2+ staged applications
// Structure: bin/<script>, lib/com.typesafe.play.play_*.jar
func (p *PlayContainer) detectPost22Staged(buildDir string) bool {
	// Check for bin/ directory at root
	binDir := filepath.Join(buildDir, "bin")
	binStat, binErr := os.Stat(binDir)
	if binErr != nil || !binStat.IsDir() {
		return false
	}

	// Check for lib/ directory at root
	libDir := filepath.Join(buildDir, "lib")
	libStat, libErr := os.Stat(libDir)
	if libErr != nil || !libStat.IsDir() {
		return false
	}

	// Find Play JAR in lib/
	playJar, version := p.findPlayJar(libDir)
	if playJar == "" {
		return false
	}

	// Parse version - must be 2.2 or higher
	if !p.isPost22Version(version) {
		return false
	}

	// Find start script in bin/
	startScript := p.findStartScript(binDir)
	if startScript == "" {
		return false
	}

	p.playType = "post22_staged"
	p.playVersion = version
	p.startScript = filepath.Join("bin", startScript)
	p.libDir = libDir
	p.context.Log.Debug("Detected Play Framework %s (Post22Staged)", version)
	return true
}

// detectPre22Dist detects Play 2.0-2.1 distributed applications
// Structure: application-root/start, application-root/lib/play_*.jar
func (p *PlayContainer) detectPre22Dist(buildDir string) bool {
	// Check for application-root/ directory
	appRoot := filepath.Join(buildDir, "application-root")
	appRootStat, err := os.Stat(appRoot)
	if err != nil || !appRootStat.IsDir() {
		return false
	}

	// Check for start script
	startScript := filepath.Join(appRoot, "start")
	if _, err := os.Stat(startScript); err != nil {
		return false
	}

	// Check for lib/ directory
	libDir := filepath.Join(appRoot, "lib")
	libStat, libErr := os.Stat(libDir)
	if libErr != nil || !libStat.IsDir() {
		return false
	}

	// Find Play JAR (play.play_*.jar or play_*.jar)
	playJar, version := p.findPlayJar(libDir)
	if playJar == "" {
		return false
	}

	// Version should be 2.0 or 2.1
	if p.isPost22Version(version) {
		return false
	}

	p.playType = "pre22_dist"
	p.playVersion = version
	p.startScript = filepath.Join("application-root", "start")
	p.libDir = libDir
	p.context.Log.Debug("Detected Play Framework %s (Pre22Dist)", version)
	return true
}

// detectPre22Staged detects Play 2.0-2.1 staged applications
// Structure: start (at root), staged/play_*.jar
func (p *PlayContainer) detectPre22Staged(buildDir string) bool {
	// Check for start script at root
	startScript := filepath.Join(buildDir, "start")
	p.context.Log.Debug("Play Pre22Staged: Checking for start script: %s", startScript)
	if _, err := os.Stat(startScript); err != nil {
		p.context.Log.Debug("Play Pre22Staged: Start script not found: %v", err)
		return false
	}
	p.context.Log.Debug("Play Pre22Staged: Start script found")

	// Check for staged/ directory
	stagedDir := filepath.Join(buildDir, "staged")
	p.context.Log.Debug("Play Pre22Staged: Checking for staged dir: %s", stagedDir)
	stagedStat, err := os.Stat(stagedDir)
	if err != nil || !stagedStat.IsDir() {
		p.context.Log.Debug("Play Pre22Staged: Staged dir not found or not a directory: %v", err)
		return false
	}
	p.context.Log.Debug("Play Pre22Staged: Staged dir found")

	// Find Play JAR in staged/
	playJar, version := p.findPlayJar(stagedDir)
	p.context.Log.Debug("Play Pre22Staged: findPlayJar returned jar=%s, version=%s", playJar, version)
	if playJar == "" {
		p.context.Log.Debug("Play Pre22Staged: No Play JAR found")
		return false
	}

	// Version should be 2.0 or 2.1
	if p.isPost22Version(version) {
		p.context.Log.Debug("Play Pre22Staged: Version %s is Post22, not Pre22", version)
		return false
	}

	p.playType = "pre22_staged"
	p.playVersion = version
	p.startScript = "start"
	p.libDir = stagedDir
	p.context.Log.Debug("Detected Play Framework %s (Pre22Staged)", version)
	return true
}

// findPlayJar finds the Play Framework JAR and extracts version
// Returns jar filename and version string
func (p *PlayContainer) findPlayJar(libDir string) (string, string) {
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return "", ""
	}

	// Match patterns:
	// - com.typesafe.play.play_2.10-2.2.0.jar (Play 2.2+)
	// - play.play_2.9.1-2.0.jar (Play 2.0)
	// - play_2.10-2.1.4.jar (Play 2.1)
	playJarPattern := regexp.MustCompile(`^(?:com\.typesafe\.)?play(?:\.play)?_.*-(.+)\.jar$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if matches := playJarPattern.FindStringSubmatch(name); matches != nil {
			version := matches[1]
			p.context.Log.Debug("Found Play JAR: %s (version: %s)", name, version)
			return name, version
		}
	}

	return "", ""
}

// findStartScript finds a non-.bat startup script in the given directory
func (p *PlayContainer) findStartScript(binDir string) string {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip .bat files
		if filepath.Ext(name) != ".bat" {
			return name
		}
	}

	return ""
}

// isPost22Version checks if version is 2.2 or higher
func (p *PlayContainer) isPost22Version(version string) bool {
	// Parse major.minor version
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	major := parts[0]
	minor := parts[1]

	// Check for 2.2+
	if major == "2" {
		// Extract numeric minor version
		minorInt := 0
		fmt.Sscanf(minor, "%d", &minorInt)
		return minorInt >= 2
	}

	// Version 3+ would also be post-2.2
	majorInt := 0
	fmt.Sscanf(major, "%d", &majorInt)
	return majorInt > 2
}

// Supply installs and configures the Play Framework application
func (p *PlayContainer) Supply() error {
	p.context.Log.BeginStep("Installing Play Framework %s (%s)", p.playVersion, p.playType)

	// Make start script executable
	if err := p.makeStartScriptExecutable(); err != nil {
		return fmt.Errorf("failed to make start script executable: %w", err)
	}

	p.context.Log.Info("Play Framework %s installation complete", p.playVersion)
	return nil
}

// makeStartScriptExecutable ensures the start script has execute permissions
func (p *PlayContainer) makeStartScriptExecutable() error {
	buildDir := p.context.Stager.BuildDir()
	scriptPath := filepath.Join(buildDir, p.startScript)

	if err := os.Chmod(scriptPath, 0755); err != nil {
		p.context.Log.Warning("Could not make %s executable: %s", p.startScript, err.Error())
		return err
	}

	p.context.Log.Debug("Made %s executable", p.startScript)
	return nil
}

// Finalize performs final configuration for the Play Framework application
func (p *PlayContainer) Finalize() error {
	p.context.Log.BeginStep("Finalizing Play Framework %s", p.playVersion)
	// Play Framework doesn't require finalization - all setup is done in Supply
	p.context.Log.Info("Play Framework finalization complete")
	return nil
}

// Release returns the command to start the Play Framework application
func (p *PlayContainer) Release() (string, error) {
	// Play Framework start command varies by type
	// All types use the start script with JAVA_OPTS environment variable

	// The start script is already relative to build directory
	cmd := p.startScript

	p.context.Log.Debug("Play Framework release command: %s", cmd)
	return cmd, nil
}

// Validate checks for ambiguous Play configurations
// This should be called during detection to reject hybrid apps
func (p *PlayContainer) Validate() error {
	buildDir := p.context.Stager.BuildDir()

	// Check for ambiguous Play 2.1/2.2 hybrid configurations
	// This happens when both Pre22 and Post22 structures exist

	detected := []string{}

	if p.detectPost22Dist(buildDir) {
		detected = append(detected, "Post22Dist")
	}
	if p.detectPost22Staged(buildDir) {
		detected = append(detected, "Post22Staged")
	}
	if p.detectPre22Dist(buildDir) {
		detected = append(detected, "Pre22Dist")
	}
	if p.detectPre22Staged(buildDir) {
		detected = append(detected, "Pre22Staged")
	}

	if len(detected) > 1 {
		return fmt.Errorf("Play Framework application version cannot be determined: %v", detected)
	}

	return nil
}
