package environment

import "fmt"

// LockParams controls how the lock file is generated.
type LockParams struct {
	Platforms []string // limit to specific platforms; empty means current host
}

// SyncParams controls how an environment is restored from a lock file.
type SyncParams struct {
	DryRun bool // resolve without applying changes
}

// Lock resolves all packages and writes index.lock.
func (e *Environment) Lock(params ...LockParams) error {
	idx, err := readIndex(e.Path)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}

	lock := &LockFile{
		Platform: make(map[string]map[string]LockedPackage),
	}

	for pkg, entry := range idx.Packages {
		key := platformKey(e.platform, entry.Platform)

		if _, ok := lock.Platform[key]; !ok {
			lock.Platform[key] = make(map[string]LockedPackage)
		}

		lock.Platform[key][pkg] = LockedPackage{
			Version:  entry.Version,
			Provider: e.platform.DefaultProvider(entry.Platform),
		}
	}

	return writeLock(e.Path, lock)
}

// Sync restores an environment from index.lock.
func (e *Environment) Sync(params ...SyncParams) error {
	dry := len(params) > 0 && params[0].DryRun

	lock, err := readLock(e.Path)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	key := platformKey(e.platform, "")
	pkgs, ok := lock.Platform[key]
	if !ok {
		return fmt.Errorf("sync: no lock entry for platform %s", key)
	}

	for pkg, locked := range pkgs {
		if dry {
			continue
		}
		if err := e.Install(pkg, InstallParams{
			Version:  locked.Version,
			Provider: locked.Provider,
		}); err != nil {
			return fmt.Errorf("sync: %w", err)
		}
	}

	return nil
}

// platformKey builds the lock file section key for a given platform target.
func platformKey(p *Platform, platform string) string {
	if platform == "" {
		return p.OS + "." + p.Arch
	}
	return p.OS + "." + p.Arch + "." + platform
}