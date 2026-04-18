package environment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const indexFile = "index.toml"

// Index is the in-memory representation of index.toml.
type Index struct {
	Env       EnvMeta                 `toml:"env"`
	Providers map[string]string       `toml:"providers,omitempty"`
	Packages  map[string]PackageEntry `toml:"packages,omitempty"`
}

// EnvMeta holds environment identity fields.
type EnvMeta struct {
	Name string `toml:"name"`
	Path string `toml:"path,omitempty"`
}

// PackageEntry is a single package declaration in index.toml.
type PackageEntry struct {
	Version  string `toml:"version,omitempty"`
	Platform string `toml:"platform,omitempty"`
}

func readIndex(envPath string) (*Index, error) {
	path := filepath.Join(envPath, indexFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx Index
	if _, err := toml.Decode(string(data), &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	return &idx, nil
}

func writeIndex(envPath string, idx *Index) error {
	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).Encode(idx); err != nil {
		return fmt.Errorf("encode index: %w", err)
	}

	path := filepath.Join(envPath, indexFile)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	return nil
}

// writeIndex bootstraps a fresh index.toml for a newly created environment.
func (e *Environment) writeIndex() error {
	return writeIndex(e.Path, &Index{
		Env: EnvMeta{
			Name: e.Name,
			Path: e.Path,
		},
	})
}