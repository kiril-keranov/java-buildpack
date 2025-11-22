package containers

import (
	"github.com/cloudfoundry/libbuildpack"
)

// Container represents a Java application container (Tomcat, Spring Boot, etc.)
type Container interface {
	// Detect returns true if this container should handle the application
	// Returns the container name and version if detected
	Detect() (string, error)

	// Supply installs the container and its dependencies
	Supply() error

	// Finalize performs final container configuration
	Finalize() error

	// Release returns the startup command for the container
	Release() (string, error)
}

// Context holds common dependencies for containers
type Context struct {
	Stager    *libbuildpack.Stager
	Manifest  *libbuildpack.Manifest
	Installer *libbuildpack.Installer
	Log       *libbuildpack.Logger
	Command   *libbuildpack.Command
}

// Registry manages available containers
type Registry struct {
	containers []Container
	context    *Context
}

// NewRegistry creates a new container registry
func NewRegistry(ctx *Context) *Registry {
	return &Registry{
		containers: []Container{},
		context:    ctx,
	}
}

// Register adds a container to the registry
func (r *Registry) Register(c Container) {
	r.containers = append(r.containers, c)
}

// Detect finds the first container that can handle the application
func (r *Registry) Detect() (Container, string, error) {
	for _, container := range r.containers {
		name, err := container.Detect()
		if err != nil {
			// Propagate errors (e.g., validation failures)
			return nil, "", err
		}
		if name != "" {
			return container, name, nil
		}
	}
	return nil, "", nil
}

// DetectAll returns all containers that can handle the application
func (r *Registry) DetectAll() ([]Container, []string, error) {
	var matched []Container
	var names []string

	for _, container := range r.containers {
		name, err := container.Detect()
		if err != nil {
			// Propagate errors (e.g., validation failures)
			return nil, nil, err
		}
		if name != "" {
			matched = append(matched, container)
			names = append(names, name)
		}
	}

	return matched, names, nil
}
