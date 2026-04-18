package environment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const lockFile = "index.lock"

// LockFile is the resolved, committed snapshot of all packages per platform.
type LockFile struct {
	Platform map[string]map[string]LockedPackage `toml:"platform"`
}

// LockedPackage is a fully resolved package entry in the lock file.
type LockedPackage struct {
	Version  string `toml:"version"`
	Provider string `toml:"provider"`
}

func readLock(envPath string) (*LockFile, error) {
	path := filepath.Join(envPath, lockFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lock: %w", err)
	}

	var lock LockFile
	if _, err := toml.Decode(string(data), &lock); err != nil {
		return nil, fmt.Errorf("parse lock: %w", err)
	}

	return &lock, nil
}

func writeLock(envPath string, lock *LockFile) error {
	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).Encode(lock); err != nil {
		return fmt.Errorf("encode lock: %w", err)
	}

	path := filepath.Join(envPath, lockFile)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}

	return nil
}