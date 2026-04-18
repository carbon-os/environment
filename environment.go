package environment

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultBaseDir = "envs"

// Environment is an isolated, self-contained package scope.
type Environment struct {
	Name     string
	Path     string
	platform *Platform
}

// CreateParams controls how a new environment is created.
type CreateParams struct {
	Path string // custom location; falls back to config base-path or default
}

// New creates a new named environment.
func New(name string, params ...CreateParams) (*Environment, error) {
	path, err := resolvePath(name, params...)
	if err != nil {
		return nil, fmt.Errorf("environment.New: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(path, "bin"), 0755); err != nil {
		return nil, fmt.Errorf("environment.New: create dirs: %w", err)
	}

	p, err := detectPlatform()
	if err != nil {
		return nil, fmt.Errorf("environment.New: detect platform: %w", err)
	}

	e := &Environment{
		Name:     name,
		Path:     path,
		platform: p,
	}

	if err := e.writeIndex(); err != nil {
		return nil, fmt.Errorf("environment.New: write index: %w", err)
	}

	return e, nil
}

// Open loads an existing environment from a path.
func Open(path string) (*Environment, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("environment.Open: %w", err)
	}

	idx, err := readIndex(abs)
	if err != nil {
		return nil, fmt.Errorf("environment.Open: %w", err)
	}

	p, err := detectPlatform()
	if err != nil {
		return nil, fmt.Errorf("environment.Open: detect platform: %w", err)
	}

	return &Environment{
		Name:     idx.Env.Name,
		Path:     abs,
		platform: p,
	}, nil
}

// BinPath returns the path to the environment's bin directory.
func (e *Environment) BinPath() string {
	return filepath.Join(e.Path, "bin")
}

func resolvePath(name string, params ...CreateParams) (string, error) {
	if len(params) > 0 && params[0].Path != "" {
		return filepath.Abs(params[0].Path)
	}

	cfg, err := loadConfig()
	if err == nil && cfg.BasePath != "" {
		return filepath.Join(cfg.BasePath, name), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".env", defaultBaseDir, name), nil
}