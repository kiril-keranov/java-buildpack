package frameworks_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry/java-buildpack/src/java/frameworks"
	"github.com/cloudfoundry/libbuildpack"
)

// Note: This file contains basic unit tests for the framework system.
// To run these tests, you need to install Ginkgo and Gomega:
//   go get github.com/onsi/ginkgo
//   go get github.com/onsi/gomega

// TestVCAPServicesHasService tests the HasService method
func TestVCAPServicesHasService(t *testing.T) {
	vcapServices := frameworks.VCAPServices{
		"newrelic": []frameworks.VCAPService{
			{Name: "newrelic-service", Label: "newrelic"},
		},
	}

	if !vcapServices.HasService("newrelic") {
		t.Error("Expected HasService to return true for 'newrelic'")
	}

	if vcapServices.HasService("appdynamics") {
		t.Error("Expected HasService to return false for 'appdynamics'")
	}
}

// TestVCAPServicesGetService tests the GetService method
func TestVCAPServicesGetService(t *testing.T) {
	vcapServices := frameworks.VCAPServices{
		"newrelic": []frameworks.VCAPService{
			{Name: "my-newrelic", Label: "newrelic"},
		},
	}

	service := vcapServices.GetService("newrelic")
	if service == nil {
		t.Fatal("Expected GetService to return a service")
	}

	if service.Name != "my-newrelic" {
		t.Errorf("Expected service name 'my-newrelic', got '%s'", service.Name)
	}

	nilService := vcapServices.GetService("appdynamics")
	if nilService != nil {
		t.Error("Expected GetService to return nil for non-existent service")
	}
}

// TestVCAPServicesHasTag tests the HasTag method
func TestVCAPServicesHasTag(t *testing.T) {
	vcapServices := frameworks.VCAPServices{
		"user-provided": []frameworks.VCAPService{
			{
				Name:  "my-monitoring",
				Label: "user-provided",
				Tags:  []string{"monitoring", "apm"},
			},
		},
	}

	if !vcapServices.HasTag("apm") {
		t.Error("Expected HasTag to return true for 'apm'")
	}

	if vcapServices.HasTag("database") {
		t.Error("Expected HasTag to return false for 'database'")
	}
}

// TestGetVCAPServicesEmpty tests parsing empty VCAP_SERVICES
func TestGetVCAPServicesEmpty(t *testing.T) {
	os.Setenv("VCAP_SERVICES", "")
	defer os.Unsetenv("VCAP_SERVICES")

	services, err := frameworks.GetVCAPServices()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(services) != 0 {
		t.Errorf("Expected empty services map, got: %d services", len(services))
	}
}

// TestGetVCAPServicesValid tests parsing valid VCAP_SERVICES JSON
func TestGetVCAPServicesValid(t *testing.T) {
	vcapJSON := `{
		"newrelic": [{
			"name": "newrelic-service",
			"label": "newrelic",
			"tags": ["apm", "monitoring"],
			"credentials": {
				"licenseKey": "test-key-123"
			}
		}]
	}`

	os.Setenv("VCAP_SERVICES", vcapJSON)
	defer os.Unsetenv("VCAP_SERVICES")

	services, err := frameworks.GetVCAPServices()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !services.HasService("newrelic") {
		t.Error("Expected to find newrelic service")
	}

	service := services.GetService("newrelic")
	if service == nil {
		t.Fatal("Expected to get newrelic service")
	}

	if service.Name != "newrelic-service" {
		t.Errorf("Expected service name 'newrelic-service', got '%s'", service.Name)
	}

	if licenseKey, ok := service.Credentials["licenseKey"].(string); !ok || licenseKey != "test-key-123" {
		t.Error("Expected licenseKey credential to be 'test-key-123'")
	}
}

// TestFrameworkRegistry tests the framework registry
func TestFrameworkRegistry(t *testing.T) {
	// Create mock context
	stager := &libbuildpack.Stager{}
	manifest := &libbuildpack.Manifest{}
	installer := &libbuildpack.Installer{}
	logger := &libbuildpack.Logger{}
	command := &libbuildpack.Command{}

	ctx := &frameworks.Context{
		Stager:    stager,
		Manifest:  manifest,
		Installer: installer,
		Log:       logger,
		Command:   command,
	}

	// Create registry and register frameworks
	registry := frameworks.NewRegistry(ctx)
	registry.Register(frameworks.NewNewRelicFramework(ctx))
	registry.Register(frameworks.NewAppDynamicsFramework(ctx))
	registry.Register(frameworks.NewDynatraceFramework(ctx))

	// Test detection with no services (should detect nothing)
	os.Unsetenv("VCAP_SERVICES")
	detected, names, err := registry.DetectAll()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(detected) != 0 {
		t.Errorf("Expected no frameworks detected, got: %v", names)
	}
}

// TestNewRelicFrameworkDetect tests New Relic framework detection
func TestNewRelicFrameworkDetect(t *testing.T) {
	// Create a temporary build directory for testing
	tmpDir, err := os.MkdirTemp("", "java-buildpack-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stager := libbuildpack.NewStager([]string{tmpDir, "", "0"}, nil, &libbuildpack.Manifest{})

	ctx := &frameworks.Context{
		Stager: stager,
		Log:    &libbuildpack.Logger{},
	}

	framework := frameworks.NewNewRelicFramework(ctx)

	// Test with no service binding
	os.Unsetenv("VCAP_SERVICES")
	name, err := framework.Detect()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if name != "" {
		t.Errorf("Expected no detection without service, got: %s", name)
	}

	// Test with New Relic service
	vcapJSON := `{
		"newrelic": [{
			"name": "newrelic-service",
			"label": "newrelic",
			"credentials": {
				"licenseKey": "test-key"
			}
		}]
	}`
	os.Setenv("VCAP_SERVICES", vcapJSON)
	defer os.Unsetenv("VCAP_SERVICES")

	name, err = framework.Detect()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if name != "New Relic Agent" {
		t.Errorf("Expected 'New Relic Agent', got: %s", name)
	}
}
