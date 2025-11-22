package detect

import (
	"fmt"
	"os"
	"path/filepath"
)

type Detector struct {
	BuildDir string
	Version  string
}

// Run performs Java app detection
func Run(d *Detector) error {
	// Check for various Java application indicators

	// 1. Check for WEB-INF directory (Servlet/WAR)
	if _, err := os.Stat(filepath.Join(d.BuildDir, "WEB-INF")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 2. Check for WAR file
	matches, err := filepath.Glob(filepath.Join(d.BuildDir, "*.war"))
	if err == nil && len(matches) > 0 {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 3. Check for pom.xml (Maven)
	if _, err := os.Stat(filepath.Join(d.BuildDir, "pom.xml")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 4. Check for build.gradle or build.gradle.kts (Gradle)
	if _, err := os.Stat(filepath.Join(d.BuildDir, "build.gradle")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}
	if _, err := os.Stat(filepath.Join(d.BuildDir, "build.gradle.kts")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 5. Check for JAR file
	matches, err = filepath.Glob(filepath.Join(d.BuildDir, "*.jar"))
	if err == nil && len(matches) > 0 {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 6. Check for BOOT-INF directory (Spring Boot)
	if _, err := os.Stat(filepath.Join(d.BuildDir, "BOOT-INF")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 7. Check for META-INF/MANIFEST.MF
	if _, err := os.Stat(filepath.Join(d.BuildDir, "META-INF", "MANIFEST.MF")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 8. Check for .class files
	found := false
	err = filepath.Walk(d.BuildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".class" {
			found = true
			return filepath.SkipAll
		}
		// Don't walk too deep
		if info.IsDir() && filepath.Dir(path) != d.BuildDir {
			relPath, _ := filepath.Rel(d.BuildDir, path)
			if len(relPath) > 100 {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if found {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 9. Check for .groovy files (Groovy scripts)
	matches, err = filepath.Glob(filepath.Join(d.BuildDir, "*.groovy"))
	if err == nil && len(matches) > 0 {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 10. Check for Play Framework (start script at root, application-root, or staged-app)
	if _, err := os.Stat(filepath.Join(d.BuildDir, "start")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}
	if _, err := os.Stat(filepath.Join(d.BuildDir, "application-root", "start")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}
	if _, err := os.Stat(filepath.Join(d.BuildDir, "staged-app", "start")); err == nil {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 11. Check for Ratpack (ratpack-core JAR)
	matches, err = filepath.Glob(filepath.Join(d.BuildDir, "application-root", "lib", "ratpack-core-*.jar"))
	if err == nil && len(matches) > 0 {
		fmt.Printf("java %s\n", d.Version)
		return nil
	}

	// 12. Check for lib directory with JARs (generic Java app structure)
	libDirGeneric := filepath.Join(d.BuildDir, "application-root", "lib")
	if _, err := os.Stat(libDirGeneric); err == nil {
		matches, _ := filepath.Glob(filepath.Join(libDirGeneric, "*.jar"))
		if len(matches) > 0 {
			fmt.Printf("java %s\n", d.Version)
			return nil
		}
	}

	// 13. Check for dist-zip structure (bin/ and lib/ directories)
	// Check at root level
	binDir := filepath.Join(d.BuildDir, "bin")
	libDir := filepath.Join(d.BuildDir, "lib")
	binStat, binErr := os.Stat(binDir)
	libStat, libErr := os.Stat(libDir)
	if binErr == nil && libErr == nil && binStat.IsDir() && libStat.IsDir() {
		// Check for startup scripts in bin/
		entries, err := os.ReadDir(binDir)
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
					fmt.Printf("java %s\n", d.Version)
					return nil
				}
			}
		}
	}

	// Check at application-root level
	binDirApp := filepath.Join(d.BuildDir, "application-root", "bin")
	libDirApp := filepath.Join(d.BuildDir, "application-root", "lib")
	binStatApp, binErrApp := os.Stat(binDirApp)
	libStatApp, libErrApp := os.Stat(libDirApp)
	if binErrApp == nil && libErrApp == nil && binStatApp.IsDir() && libStatApp.IsDir() {
		// Check for startup scripts in bin/
		entriesApp, errApp := os.ReadDir(binDirApp)
		if errApp == nil && len(entriesApp) > 0 {
			for _, entry := range entriesApp {
				if !entry.IsDir() && filepath.Ext(entry.Name()) != ".bat" {
					fmt.Printf("java %s\n", d.Version)
					return nil
				}
			}
		}
	}

	// 14. Check for Procfile with java command
	procfilePath := filepath.Join(d.BuildDir, "Procfile")
	if data, err := os.ReadFile(procfilePath); err == nil {
		content := string(data)
		if len(content) > 0 {
			// Simple check for java in Procfile
			// In a more complete implementation, we'd parse the Procfile properly
			fmt.Printf("java %s\n", d.Version)
			return nil
		}
	}

	// No Java app detected
	return fmt.Errorf("no Java app detected")
}
