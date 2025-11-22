package frameworks

import (
	"encoding/json"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

// Framework represents a cross-cutting concern (APM agents, security providers, etc.)
type Framework interface {
	// Detect returns true if this framework should be included
	// Returns the framework name and version if detected
	Detect() (string, error)

	// Supply installs the framework
	Supply() error

	// Finalize performs final framework configuration
	Finalize() error
}

// Context holds common dependencies for frameworks
type Context struct {
	Stager    *libbuildpack.Stager
	Manifest  *libbuildpack.Manifest
	Installer *libbuildpack.Installer
	Log       *libbuildpack.Logger
	Command   *libbuildpack.Command
}

// Registry manages available frameworks
type Registry struct {
	frameworks []Framework
	context    *Context
}

// NewRegistry creates a new framework registry
func NewRegistry(ctx *Context) *Registry {
	return &Registry{
		frameworks: []Framework{},
		context:    ctx,
	}
}

// Register adds a framework to the registry
func (r *Registry) Register(f Framework) {
	r.frameworks = append(r.frameworks, f)
}

// DetectAll returns all frameworks that should be included
func (r *Registry) DetectAll() ([]Framework, []string, error) {
	var matched []Framework
	var names []string

	for _, framework := range r.frameworks {
		if name, err := framework.Detect(); err == nil && name != "" {
			matched = append(matched, framework)
			names = append(names, name)
		}
	}

	return matched, names, nil
}

// VCAPServices represents the VCAP_SERVICES environment variable structure
type VCAPServices map[string][]VCAPService

// VCAPService represents a single service binding
type VCAPService struct {
	Name        string                 `json:"name"`
	Label       string                 `json:"label"`
	Tags        []string               `json:"tags"`
	Credentials map[string]interface{} `json:"credentials"`
}

// GetVCAPServices parses the VCAP_SERVICES environment variable
func GetVCAPServices() (VCAPServices, error) {
	vcapServicesStr := os.Getenv("VCAP_SERVICES")
	if vcapServicesStr == "" {
		return VCAPServices{}, nil
	}

	var services VCAPServices
	if err := json.Unmarshal([]byte(vcapServicesStr), &services); err != nil {
		return nil, err
	}

	return services, nil
}

// HasService checks if a service with the given label exists
func (v VCAPServices) HasService(label string) bool {
	_, exists := v[label]
	return exists
}

// GetService returns the first service with the given label
func (v VCAPServices) GetService(label string) *VCAPService {
	services, exists := v[label]
	if !exists || len(services) == 0 {
		return nil
	}
	return &services[0]
}

// HasTag checks if any service has the given tag
func (v VCAPServices) HasTag(tag string) bool {
	for _, serviceList := range v {
		for _, service := range serviceList {
			for _, t := range service.Tags {
				if t == tag {
					return true
				}
			}
		}
	}
	return false
}
