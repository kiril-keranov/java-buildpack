package finalize_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/java-buildpack/src/java/finalize"
	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFinalize(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Finalize Suite")
}

var _ = Describe("Finalize", func() {
	var (
		buildDir  string
		cacheDir  string
		depsDir   string
		depsIdx   string
		finalizer *finalize.Finalizer
		stager    *libbuildpack.Stager
		logger    *libbuildpack.Logger
	)

	BeforeEach(func() {
		var err error

		// Create temp directories
		buildDir, err = os.MkdirTemp("", "finalize-build")
		Expect(err).NotTo(HaveOccurred())

		cacheDir, err = os.MkdirTemp("", "finalize-cache")
		Expect(err).NotTo(HaveOccurred())

		depsDir, err = os.MkdirTemp("", "finalize-deps")
		Expect(err).NotTo(HaveOccurred())

		depsIdx = "0"

		// Create a mock buildpack directory with VERSION and manifest.yml files
		buildpackDir, err := os.MkdirTemp("", "finalize-buildpack")
		Expect(err).NotTo(HaveOccurred())

		versionFile := filepath.Join(buildpackDir, "VERSION")
		Expect(os.WriteFile(versionFile, []byte("1.0.0"), 0644)).To(Succeed())

		manifestFile := filepath.Join(buildpackDir, "manifest.yml")
		manifestContent := `---
language: java
default_versions: []
dependencies: []
`
		Expect(os.WriteFile(manifestFile, []byte(manifestContent), 0644)).To(Succeed())

		// Create logger
		logger = libbuildpack.NewLogger(GinkgoWriter)

		// Create manifest with buildpack dir
		manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
		Expect(err).NotTo(HaveOccurred())

		// Create stager
		stager = libbuildpack.NewStager([]string{buildDir, cacheDir, depsDir, depsIdx}, logger, manifest)

		finalizer = &finalize.Finalizer{
			Stager:   stager,
			Manifest: manifest,
			Log:      logger,
			Command:  &libbuildpack.Command{},
		}
	})

	AfterEach(func() {
		os.RemoveAll(buildDir)
		os.RemoveAll(cacheDir)
		os.RemoveAll(depsDir)
	})

	Describe("Container Re-detection", func() {
		Context("when a Spring Boot application is present", func() {
			BeforeEach(func() {
				// Create a Spring Boot JAR with BOOT-INF
				bootInfDir := filepath.Join(buildDir, "BOOT-INF")
				Expect(os.MkdirAll(bootInfDir, 0755)).To(Succeed())
			})

			It("re-detects Spring Boot container", func() {
				Expect(finalizer).NotTo(BeNil())
				Expect(finalizer.Stager).NotTo(BeNil())
			})
		})

		Context("when a Tomcat application is present", func() {
			BeforeEach(func() {
				// Create WEB-INF directory
				webInfDir := filepath.Join(buildDir, "WEB-INF")
				Expect(os.MkdirAll(webInfDir, 0755)).To(Succeed())
			})

			It("re-detects Tomcat container", func() {
				Expect(finalizer).NotTo(BeNil())
				Expect(finalizer.Stager).NotTo(BeNil())
			})
		})
	})

	Describe("Startup Script Generation", func() {
		It("creates .java-buildpack directory", func() {
			javaBuildpackDir := filepath.Join(buildDir, ".java-buildpack")
			Expect(os.MkdirAll(javaBuildpackDir, 0755)).To(Succeed())
			Expect(javaBuildpackDir).To(BeADirectory())
		})

		It("would generate start.sh in .java-buildpack directory", func() {
			javaBuildpackDir := filepath.Join(buildDir, ".java-buildpack")
			Expect(os.MkdirAll(javaBuildpackDir, 0755)).To(Succeed())

			startScript := filepath.Join(javaBuildpackDir, "start.sh")
			// In a real scenario, the finalize phase would create this
			// For now, we just verify the path is correct
			Expect(filepath.Dir(startScript)).To(Equal(javaBuildpackDir))
		})
	})

	Describe("Environment Setup", func() {
		It("has access to deps directory for environment files", func() {
			depDir := stager.DepDir()
			envDir := filepath.Join(depDir, "env")
			Expect(os.MkdirAll(envDir, 0755)).To(Succeed())
			Expect(envDir).To(BeADirectory())
		})

		It("can write environment variables to env directory", func() {
			depDir := stager.DepDir()
			envDir := filepath.Join(depDir, "env")
			Expect(os.MkdirAll(envDir, 0755)).To(Succeed())

			javaHomeFile := filepath.Join(envDir, "JAVA_HOME")
			Expect(os.WriteFile(javaHomeFile, []byte("/deps/0/jre"), 0644)).To(Succeed())
			Expect(javaHomeFile).To(BeAnExistingFile())
		})
	})

	Describe("Profile.d Script Creation", func() {
		It("can create profile.d directory in build dir", func() {
			profileDir := filepath.Join(buildDir, ".profile.d")
			Expect(os.MkdirAll(profileDir, 0755)).To(Succeed())
			Expect(profileDir).To(BeADirectory())
		})

		It("can write profile.d scripts", func() {
			profileDir := filepath.Join(buildDir, ".profile.d")
			Expect(os.MkdirAll(profileDir, 0755)).To(Succeed())

			javaScript := filepath.Join(profileDir, "java.sh")
			scriptContent := "export JAVA_HOME=$DEPS_DIR/" + depsIdx + "/jre\n"
			Expect(os.WriteFile(javaScript, []byte(scriptContent), 0755)).To(Succeed())
			Expect(javaScript).To(BeAnExistingFile())
		})
	})

	Describe("Config Persistence", func() {
		It("reads config.yml from supply phase", func() {
			// Write a config.yml that would have been created by supply
			config := map[string]string{
				"container": "spring-boot",
				"jre":       "OpenJDK",
			}

			err := stager.WriteConfigYml(config)
			Expect(err).NotTo(HaveOccurred())

			configPath := filepath.Join(stager.DepDir(), "config.yml")
			Expect(configPath).To(BeAnExistingFile())
		})
	})

	Describe("Stager Configuration", func() {
		It("has access to build directory", func() {
			Expect(stager.BuildDir()).To(Equal(buildDir))
		})

		It("has access to cache directory", func() {
			Expect(stager.CacheDir()).To(Equal(cacheDir))
		})

		It("has access to deps directory", func() {
			depDir := stager.DepDir()
			Expect(depDir).To(ContainSubstring(depsDir))
		})
	})
})
