package containers

import (
	"fmt"
	"github.com/cloudfoundry/java-buildpack/src/java/common"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PlayContainer represents a Play Framework application container
type PlayContainer struct {
	context     *common.Context
	playType    string // "pre22_dist", "pre22_staged", "post22_dist", "post22_staged"
	playVersion string
	startScript string // relative path from buildDir
	libDir      string // absolute staging path
	appRoot     string // absolute staging path to the app root (may equal buildDir for staged apps)
}

// NewPlayContainer creates a new Play Framework container
func NewPlayContainer(ctx *common.Context) *PlayContainer {
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

	// Try to detect Play Framework type in order of specificity.
	// Staged apps are checked before dist apps (more specific structure).

	// 1. Pre22Staged (Play 2.0-2.1 staged: staged/ dir with play JARs)
	p.context.Log.Debug("Play: Trying Pre22Staged detection")
	if p.detectPre22Staged(buildDir) {
		p.context.Log.Info("Play: Detected Pre22Staged - version %s", p.playVersion)
		return "Play", nil
	}

	// 2. Post22Staged (Play 2.2+ staged: lib/ dir with play JARs at root)
	p.context.Log.Debug("Play: Trying Post22Staged detection")
	if p.detectPost22Staged(buildDir) {
		p.context.Log.Info("Play: Detected Post22Staged - version %s", p.playVersion)
		return "Play", nil
	}

	// 3. Post22Dist (Play 2.2+ dist: application-root/bin/ + application-root/lib/)
	p.context.Log.Debug("Play: Trying Post22Dist detection")
	if p.detectPost22Dist(buildDir) {
		p.context.Log.Info("Play: Detected Post22Dist - version %s", p.playVersion)
		return "Play", nil
	}

	// 4. Pre22Dist (Play 2.0-2.1 dist: application-root/start + application-root/lib/)
	p.context.Log.Debug("Play: Trying Pre22Dist detection")
	if p.detectPre22Dist(buildDir) {
		p.context.Log.Info("Play: Detected Pre22Dist - version %s", p.playVersion)
		return "Play", nil
	}

	p.context.Log.Debug("Play: No Play Framework detected")
	return "", nil
}

// detectPost22Dist detects Play 2.2+ distributed applications.
// Structure: application-root/bin/<script>, application-root/lib/com.typesafe.play.play_*.jar
func (p *PlayContainer) detectPost22Dist(buildDir string) bool {
	appRoot := filepath.Join(buildDir, "application-root")
	binDir := filepath.Join(appRoot, "bin")
	libDir := filepath.Join(appRoot, "lib")

	if !isDir(binDir) || !isDir(libDir) {
		return false
	}

	playJar, version := p.findPlayJar(libDir)
	if playJar == "" || !p.isPost22Version(version) {
		return false
	}

	startScript := p.findStartScript(binDir)
	if startScript == "" {
		return false
	}

	p.playType = "post22_dist"
	p.playVersion = version
	p.startScript = filepath.Join("application-root", "bin", startScript)
	p.libDir = libDir
	p.appRoot = appRoot
	p.context.Log.Debug("Detected Play Framework %s (Post22Dist)", version)
	return true
}

// detectPost22Staged detects Play 2.2+ staged applications.
// Structure: lib/com.typesafe.play.play_*.jar at the build root, optional bin/<script>
func (p *PlayContainer) detectPost22Staged(buildDir string) bool {
	libDir := filepath.Join(buildDir, "lib")
	if !isDir(libDir) {
		return false
	}

	playJar, version := p.findPlayJar(libDir)
	if playJar == "" || !p.isPost22Version(version) {
		return false
	}

	startScript := ""
	binDir := filepath.Join(buildDir, "bin")
	if isDir(binDir) {
		if s := p.findStartScript(binDir); s != "" {
			startScript = filepath.Join("bin", s)
		}
	}

	p.playType = "post22_staged"
	p.playVersion = version
	p.startScript = startScript
	p.libDir = libDir
	p.appRoot = buildDir
	p.context.Log.Debug("Detected Play Framework %s (Post22Staged)", version)
	return true
}

// detectPre22Dist detects Play 2.0-2.1 distributed applications.
// Structure: application-root/start, application-root/lib/play_*.jar
func (p *PlayContainer) detectPre22Dist(buildDir string) bool {
	appRoot := filepath.Join(buildDir, "application-root")
	if !isDir(appRoot) {
		return false
	}

	startScriptPath := filepath.Join(appRoot, "start")
	if _, err := os.Stat(startScriptPath); err != nil {
		return false
	}

	libDir := filepath.Join(appRoot, "lib")
	if !isDir(libDir) {
		return false
	}

	playJar, version := p.findPlayJar(libDir)
	if playJar == "" || p.isPost22Version(version) {
		return false
	}

	p.playType = "pre22_dist"
	p.playVersion = version
	p.startScript = filepath.Join("application-root", "start")
	p.libDir = libDir
	p.appRoot = appRoot
	p.context.Log.Debug("Detected Play Framework %s (Pre22Dist)", version)
	return true
}

// detectPre22Staged detects Play 2.0-2.1 staged applications.
// Structure: staged/play_*.jar, optional start script at root
func (p *PlayContainer) detectPre22Staged(buildDir string) bool {
	stagedDir := filepath.Join(buildDir, "staged")
	p.context.Log.Debug("Play Pre22Staged: Checking for staged dir: %s", stagedDir)
	if !isDir(stagedDir) {
		p.context.Log.Debug("Play Pre22Staged: Staged dir not found")
		return false
	}

	playJar, version := p.findPlayJar(stagedDir)
	p.context.Log.Debug("Play Pre22Staged: findPlayJar returned jar=%s, version=%s", playJar, version)
	if playJar == "" || p.isPost22Version(version) {
		p.context.Log.Debug("Play Pre22Staged: No suitable Play JAR found")
		return false
	}

	startScript := ""
	if _, err := os.Stat(filepath.Join(buildDir, "start")); err == nil {
		startScript = "start"
	}

	p.playType = "pre22_staged"
	p.playVersion = version
	p.startScript = startScript
	p.libDir = stagedDir
	p.appRoot = buildDir
	p.context.Log.Debug("Detected Play Framework %s (Pre22Staged)", version)
	return true
}

// Supply makes start scripts executable and augments the classpath in the script.
//
// Ruby buildpack compile() behaviour (base.rb):
//  1. Replace play.core.server.NettyServer → org.cloudfoundry.reconfiguration.play.Bootstrap
//  2. chmod 0755 the start script
//  3. augment_classpath: prepend additional library paths into the script's classpath declaration
func (p *PlayContainer) Supply() error {
	p.context.Log.BeginStep("Installing Play Framework %s (%s)", p.playVersion, p.playType)

	if p.startScript == "" {
		p.context.Log.Info("No start script found, skipping script modifications")
		return nil
	}

	buildDir := p.context.Stager.BuildDir()
	scriptPath := filepath.Join(buildDir, p.startScript)

	// 1. Replace the bootstrap class (base.rb: update_file start_script, ORIGINAL_BOOTSTRAP, REPLACEMENT_BOOTSTRAP)
	if err := p.replaceInFile(scriptPath,
		"play.core.server.NettyServer",
		"org.cloudfoundry.reconfiguration.play.Bootstrap"); err != nil {
		p.context.Log.Warning("Could not replace bootstrap class in %s: %s", p.startScript, err.Error())
	}

	// 2. chmod 0755
	if err := os.Chmod(scriptPath, 0755); err != nil {
		p.context.Log.Warning("Could not make %s executable: %s", p.startScript, err.Error())
	}

	// 3. Augment classpath in the start script with additional library paths
	additionalLibs := p.collectAdditionalLibraries()
	p.context.Log.Info("Found %d additional libraries for classpath augmentation", len(additionalLibs))
	if len(additionalLibs) > 0 {
		if err := p.augmentClasspath(scriptPath, additionalLibs); err != nil {
			p.context.Log.Warning("Could not augment classpath in %s: %s", p.startScript, err.Error())
		}
	}

	p.context.Log.Info("Play Framework %s installation complete", p.playVersion)
	return nil
}

// augmentClasspath prepends additional library paths into the start script's classpath declaration.
//
// Post-2.2 scripts (post22.rb):
//
//	Replaces:  declare -r app_classpath="<existing>"
//	With:      declare -r app_classpath="$app_home/<rel1>:$app_home/<rel2>:<existing>"
//
// Pre-2.2 dist scripts (pre22_dist.rb, Play 2.1):
//
//	Replaces:  classpath="<existing>"
//	With:      classpath="$scriptdir/<rel1>:$scriptdir/<rel2>:<existing>"
//
// Pre-2.2 dist scripts (pre22_dist.rb, Play 2.0) and Pre-2.2 staged:
//
//	Symlinks additional libraries directly into the lib directory (link_to).
//	In Go we copy the runtime paths directly into the script environment via profile.d instead.
func (p *PlayContainer) augmentClasspath(scriptPath string, additionalLibs []string) error {
	switch p.playType {
	case "post22_dist", "post22_staged":
		return p.augmentPost22Classpath(scriptPath, additionalLibs)
	case "pre22_dist":
		if p.playVersion != "" && strings.HasPrefix(p.playVersion, "2.0") {
			// Play 2.0: link_to behaviour — handled via profile.d CLASSPATH, nothing to do in the script
			return nil
		}
		return p.augmentPre22DistClasspath(scriptPath, additionalLibs)
	case "pre22_staged":
		// link_to behaviour — handled via profile.d CLASSPATH
		return nil
	}
	return nil
}

// augmentPost22Classpath prepends additional libraries to the `declare -r app_classpath="..."` line.
// Ruby: update_file start_script, /^declare -r app_classpath="(.*)"$/, "declare -r app_classpath=\"#{additional}:\\1\""
func (p *PlayContainer) augmentPost22Classpath(scriptPath string, additionalLibs []string) error {
	scriptDir := filepath.Dir(scriptPath)
	var classpathEntries []string
	for _, lib := range additionalLibs {
		rel, err := filepath.Rel(scriptDir, lib)
		if err != nil {
			rel = lib
		}
		classpathEntries = append(classpathEntries, "$app_home/"+filepath.ToSlash(rel))
	}

	prefix := strings.Join(classpathEntries, ":")
	pattern := regexp.MustCompile(`(?m)^(declare -r app_classpath=")(.*)(")\s*$`)
	return p.replaceInFileRegexp(scriptPath, pattern, "${1}"+prefix+":${2}${3}")
}

// augmentPre22DistClasspath prepends additional libraries to the `classpath="..."` line.
// Ruby: update_file start_script, /^classpath="(.*)"$/, "classpath=\"#{additional}:\\1\""
func (p *PlayContainer) augmentPre22DistClasspath(scriptPath string, additionalLibs []string) error {
	scriptDir := filepath.Dir(scriptPath)
	var classpathEntries []string
	for _, lib := range additionalLibs {
		rel, err := filepath.Rel(scriptDir, lib)
		if err != nil {
			rel = lib
		}
		classpathEntries = append(classpathEntries, "$scriptdir/"+filepath.ToSlash(rel))
	}

	prefix := strings.Join(classpathEntries, ":")
	pattern := regexp.MustCompile(`(?m)^(classpath=")(.*)(")\s*$`)
	return p.replaceInFileRegexp(scriptPath, pattern, "${1}"+prefix+":${2}${3}")
}

// Finalize writes the http.port system property to JAVA_OPTS.
// (Ruby base.rb release(): @droplet.java_opts.add_system_property 'http.port', '$PORT')
func (p *PlayContainer) Finalize() error {
	p.context.Log.BeginStep("Finalizing Play Framework %s", p.playVersion)

	if err := p.context.Stager.WriteEnvFile("JAVA_OPTS", "-Dhttp.port=$PORT"); err != nil {
		return fmt.Errorf("failed to write JAVA_OPTS: %w", err)
	}

	p.context.Log.Info("Play Framework finalization complete")
	return nil
}

// Release returns the command to start the Play Framework application.
//
// Ruby base.rb release():
//
//	exec <start_script> <java_opts>
//
// Post-2.2 java_opts (post22.rb):  $(for I in $JAVA_OPTS ; do echo "-J$I" ; done)
// Pre-2.2 java_opts  (pre22.rb):   $JAVA_OPTS
func (p *PlayContainer) Release() (string, error) {
	if p.playType == "" {
		return "", fmt.Errorf("no Play application detected, Detect() must be called first")
	}

	if p.startScript != "" {
		// exec <path> <java_opts>
		// Ruby: qualify_path(start_script, @droplet.root) → at runtime this is $HOME/<relative>
		scriptCmd := "$HOME/" + filepath.ToSlash(p.startScript)
		javaOpts := p.javaOptsExpression()
		return fmt.Sprintf("exec %s %s", scriptCmd, javaOpts), nil
	}

	// No start script — fall back to direct java invocation (staged apps without a script)
	relLib, err := filepath.Rel(p.context.Stager.BuildDir(), p.libDir)
	if err != nil {
		relLib = p.libDir
	}
	relLib = filepath.ToSlash(relLib)
	return fmt.Sprintf("eval exec java $JAVA_OPTS -cp $HOME/%s/* play.core.server.NettyServer $HOME", relLib), nil
}

// javaOptsExpression returns the shell expression used to pass JAVA_OPTS to the start script.
// Post-2.2: $(for I in $JAVA_OPTS ; do echo "-J$I" ; done)   (post22.rb)
// Pre-2.2:  $JAVA_OPTS                                         (pre22.rb)
func (p *PlayContainer) javaOptsExpression() string {
	if strings.HasPrefix(p.playType, "post22") {
		return `$(for I in $JAVA_OPTS ; do echo "-J$I" ; done)`
	}
	return "$JAVA_OPTS"
}

// Validate rejects applications that match more than one Play variant (ambiguous).
// Ruby factory.rb: raise if candidates.size > 1
func (p *PlayContainer) Validate() error {
	buildDir := p.context.Stager.BuildDir()

	var detected []string
	probe := &PlayContainer{context: p.context}

	if probe.detectPost22Dist(buildDir) {
		detected = append(detected, "Post22Dist")
	}
	probe = &PlayContainer{context: p.context}
	if probe.detectPost22Staged(buildDir) {
		detected = append(detected, "Post22Staged")
	}
	probe = &PlayContainer{context: p.context}
	if probe.detectPre22Dist(buildDir) {
		detected = append(detected, "Pre22Dist")
	}
	probe = &PlayContainer{context: p.context}
	if probe.detectPre22Staged(buildDir) {
		detected = append(detected, "Pre22Staged")
	}

	if len(detected) > 1 {
		return fmt.Errorf("Play Framework application version cannot be determined: %v", detected)
	}
	return nil
}

// ---- helpers ----------------------------------------------------------------

// findPlayJar finds the Play Framework JAR in a directory and returns (filename, version).
// Matches: com.typesafe.play.play_*-<version>.jar  (Post-2.2)
//
//	play.play_*-<version>.jar                   (Play 2.0)
//	play_*-<version>.jar                        (Play 2.1)
func (p *PlayContainer) findPlayJar(libDir string) (string, string) {
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return "", ""
	}

	// Ruby base.rb: (lib_dir + '*play_*-*.jar').glob.first
	playJarPattern := regexp.MustCompile(`^(?:com\.typesafe\.)?play(?:\.play)?_.*-(.+)\.jar$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if matches := playJarPattern.FindStringSubmatch(entry.Name()); matches != nil {
			version := matches[1]
			p.context.Log.Debug("Found Play JAR: %s (version: %s)", entry.Name(), version)
			return entry.Name(), version
		}
	}
	return "", ""
}

// findStartScript returns the name of the first non-.bat file in binDir.
func (p *PlayContainer) findStartScript(binDir string) string {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
			return entry.Name()
		}
	}
	return ""
}

// isPost22Version returns true when version is 2.2 or higher.
func (p *PlayContainer) isPost22Version(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	var major, minor int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	return major > 2 || (major == 2 && minor >= 2)
}

// collectAdditionalLibraries returns all JAR paths installed under DepDir by supply buildpacks.
func (p *PlayContainer) collectAdditionalLibraries() []string {
	var libs []string
	depsDir := p.context.Stager.DepDir()

	entries, err := os.ReadDir(depsDir)
	if err != nil {
		p.context.Log.Debug("Unable to read deps directory: %s", err.Error())
		return libs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(depsDir, entry.Name(), "*.jar"))
		if err != nil {
			continue
		}
		libs = append(libs, matches...)
	}
	return libs
}

// replaceInFile does a literal string replacement inside a file.
// Ruby base.rb: update_file(path, pattern, replacement)
func (p *PlayContainer) replaceInFile(path, old, newStr string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated := strings.ReplaceAll(string(data), old, newStr)
	return os.WriteFile(path, []byte(updated), 0644)
}

// replaceInFileRegexp does a regexp replacement inside a file.
func (p *PlayContainer) replaceInFileRegexp(path string, pattern *regexp.Regexp, replacement string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated := pattern.ReplaceAllString(string(data), replacement)
	return os.WriteFile(path, []byte(updated), 0644)
}
