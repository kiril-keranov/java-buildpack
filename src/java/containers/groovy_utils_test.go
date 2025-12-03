package containers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasMainMethod(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has static void main",
			content: `class MyApp {
	static void main(String[] args) {
		println "Hello"
	}
}`,
			expected: true,
		},
		{
			name: "has static void main with whitespace variations",
			content: `class MyApp {
	static  void  main ( String[] args ) {
		println "Hello"
	}
}`,
			expected: true,
		},
		{
			name: "no main method",
			content: `class Alpha {
}`,
			expected: false,
		},
		{
			name:     "simple script no main",
			content:  `println 'Hello World'`,
			expected: false,
		},
		{
			name: "instance method not static main",
			content: `class Test {
	void main() {
		println "Not static"
	}
}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.groovy")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()

			result, err := HasMainMethod(tmpFile.Name())
			if err != nil {
				t.Fatalf("HasMainMethod() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("HasMainMethod() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsPOGO(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "simple class definition",
			content: `class Alpha {
}`,
			expected: true,
		},
		{
			name: "class with inheritance",
			content: `class MyApp extends BaseApp {
	void run() {}
}`,
			expected: true,
		},
		{
			name:     "simple script no class",
			content:  `println 'Hello World'`,
			expected: false,
		},
		{
			name: "script with variables no class",
			content: `def name = "World"
println "Hello $name"`,
			expected: false,
		},
		{
			name: "class keyword in comment",
			content: `// This is not a class
println 'Hello'`,
			expected: false,
		},
		{
			name:     "class keyword in string",
			content:  `println "This mentions class but isn't one"`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.groovy")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()

			result, err := IsPOGO(tmpFile.Name())
			if err != nil {
				t.Fatalf("IsPOGO() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("IsPOGO() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasShebang(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has shebang",
			content: `#!/usr/bin/env groovy
println 'Hello World'`,
			expected: true,
		},
		{
			name: "has groovy shebang",
			content: `#!/usr/bin/groovy
println 'Hello'`,
			expected: true,
		},
		{
			name: "no shebang",
			content: `class Alpha {
}`,
			expected: false,
		},
		{
			name: "shebang not at start",
			content: `
#!/usr/bin/env groovy
println 'Hello'`,
			expected: false,
		},
		{
			name: "comment mentioning shebang",
			content: `// Use #!/usr/bin/env groovy at the top
println 'Hello'`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.groovy")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()

			result, err := HasShebang(tmpFile.Name())
			if err != nil {
				t.Fatalf("HasShebang() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("HasShebang() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMainGroovyScript(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "groovy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	pogoFile := filepath.Join(tmpDir, "Alpha.groovy")
	if err := os.WriteFile(pogoFile, []byte("class Alpha {}"), 0644); err != nil {
		t.Fatal(err)
	}

	nonPogoFile := filepath.Join(tmpDir, "Application.groovy")
	if err := os.WriteFile(nonPogoFile, []byte("println 'Hello World'"), 0644); err != nil {
		t.Fatal(err)
	}

	mainMethodFile := filepath.Join(tmpDir, "Main.groovy")
	mainContent := `class Main {
	static void main(String[] args) {
		println "Main"
	}
}`
	if err := os.WriteFile(mainMethodFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	shebangFile := filepath.Join(tmpDir, "Script.groovy")
	if err := os.WriteFile(shebangFile, []byte("#!/usr/bin/env groovy\nprintln 'Script'"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		scripts  []string
		expected string
	}{
		{
			name:     "single non-POGO script",
			scripts:  []string{nonPogoFile},
			expected: nonPogoFile,
		},
		{
			name:     "POGO and non-POGO - selects non-POGO",
			scripts:  []string{pogoFile, nonPogoFile},
			expected: nonPogoFile,
		},
		{
			name:     "single file with main method",
			scripts:  []string{mainMethodFile},
			expected: mainMethodFile,
		},
		{
			name:     "single file with shebang",
			scripts:  []string{shebangFile},
			expected: shebangFile,
		},
		{
			name:     "only POGO - no candidate",
			scripts:  []string{pogoFile},
			expected: "",
		},
		{
			name: "multiple candidates - returns empty",
			// Both non-POGO and shebang file are candidates
			scripts:  []string{nonPogoFile, shebangFile},
			expected: "",
		},
		{
			name:     "empty list",
			scripts:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindMainGroovyScript(tt.scripts)
			if err != nil {
				t.Fatalf("FindMainGroovyScript() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("FindMainGroovyScript() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMainGroovyScriptWithInvalidFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "groovy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an unreadable file (binary garbage)
	invalidFile := filepath.Join(tmpDir, "invalid.groovy")
	if err := os.WriteFile(invalidFile, []byte{0xff, 0xfe}, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid non-POGO file
	validFile := filepath.Join(tmpDir, "valid.groovy")
	if err := os.WriteFile(validFile, []byte("println 'Hello'"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should skip invalid file and select valid one
	scripts := []string{invalidFile, validFile}
	result, err := FindMainGroovyScript(scripts)
	if err != nil {
		t.Fatalf("FindMainGroovyScript() error = %v", err)
	}
	if result != validFile {
		t.Errorf("FindMainGroovyScript() = %v, want %v", result, validFile)
	}
}
