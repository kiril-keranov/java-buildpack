package containers

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// GroovyUtils provides utilities for analyzing Groovy files
type GroovyUtils struct{}

var (
	// Regex patterns for Groovy file analysis
	beansPattern      = regexp.MustCompile(`beans\s*\{`)
	mainMethodPattern = regexp.MustCompile(`static\s+void\s+main\s*\(`)
	pogoPattern       = regexp.MustCompile(`class\s+\w+[\s\w]*\{`)
	shebangPattern    = regexp.MustCompile(`^#!`)
)

// FindGroovyFiles finds all .groovy files in the build directory
func (g *GroovyUtils) FindGroovyFiles(buildDir string) ([]string, error) {
	var groovyFiles []string

	err := filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for .groovy extension
		if filepath.Ext(path) == ".groovy" {
			groovyFiles = append(groovyFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return groovyFiles, nil
}

// IsBeans checks if a Groovy file contains beans { } configuration
func (g *GroovyUtils) IsBeans(filePath string) bool {
	content, err := g.readFile(filePath)
	if err != nil {
		return false
	}

	return beansPattern.MatchString(content)
}

// HasMainMethod checks if a Groovy file contains a main() method
func (g *GroovyUtils) HasMainMethod(filePath string) bool {
	content, err := g.readFile(filePath)
	if err != nil {
		return false
	}

	return mainMethodPattern.MatchString(content)
}

// IsPOGO checks if a Groovy file is a Plain Old Groovy Object (contains class definition)
func (g *GroovyUtils) IsPOGO(filePath string) bool {
	content, err := g.readFile(filePath)
	if err != nil {
		return false
	}

	return pogoPattern.MatchString(content)
}

// HasShebang checks if a Groovy file has a shebang (#!)
func (g *GroovyUtils) HasShebang(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		return shebangPattern.MatchString(firstLine)
	}

	return false
}

// IsLogbackConfigFile checks if the file is a Logback configuration file
func (g *GroovyUtils) IsLogbackConfigFile(filePath string) bool {
	return regexp.MustCompile(`ch/qos/logback/.*\.groovy$`).MatchString(filePath)
}

// readFile safely reads a file and returns its content
func (g *GroovyUtils) readFile(filePath string) (string, error) {
	// Check if file exists and is readable
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// Skip very large files (> 10MB) to avoid memory issues
	if info.Size() > 10*1024*1024 {
		return "", nil
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
