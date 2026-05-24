package environment

import (
	"fmt"

	"github.com/carbon-os/environment/provider/vcpkg"
)

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

	// Resolve the vcpkg commit once upfront — only paid if the environment
	// actually contains vcpkg-managed packages.
	vcpkgCommit, err := e.resolveVcpkgCommit(idx)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}

	lock := &LockFile{
		Platform: make(map[string]map[string]LockedPackage),
	}

	for pkg, entry := range idx.Packages {
		provider := entry.Provider
		if provider == "" {
			provider = e.platform.DefaultProvider(entry.Platform)
		}

		var (
			key    string
			locked LockedPackage
		)

		if provider == "vcpkg" {
			triplet, err := vcpkg.ResolveTriplet(entry.Platform)
			if err != nil {
				return fmt.Errorf("lock: %s: %w", pkg, err)
			}
			key = e.platform.OS + "." + e.platform.Arch + ".vcpkg"
			locked = LockedPackage{
				Version:     entry.Version,
				Provider:    "vcpkg",
				Triplet:     triplet,
				VcpkgCommit: vcpkgCommit,
			}
		} else {
			key = platformKey(e.platform, entry.Platform)
			locked = LockedPackage{
				Version:  entry.Version,
				Provider: provider,
			}
		}

		if _, ok := lock.Platform[key]; !ok {
			lock.Platform[key] = make(map[string]LockedPackage)
		}
		lock.Platform[key][pkg] = locked
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

	// Merge both the native and vcpkg sections so a single Sync restores
	// everything in the lock file regardless of provider.
	allPkgs := make(map[string]LockedPackage)
	for k, v := range lock.Platform[platformKey(e.platform, "")] {
		allPkgs[k] = v
	}
	for k, v := range lock.Platform[e.platform.OS+"."+e.platform.Arch+".vcpkg"] {
		allPkgs[k] = v
	}

	if len(allPkgs) == 0 {
		return fmt.Errorf("sync: no lock entries for platform %s", platformKey(e.platform, ""))
	}

	for pkg, locked := range allPkgs {
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

// resolveVcpkgCommit returns the short HEAD commit of the vcpkg clone embedded
// in this environment, or an empty string if no vcpkg packages are declared.
func (e *Environment) resolveVcpkgCommit(idx *Index) (string, error) {
	hasVcpkg := false
	for _, entry := range idx.Packages {
		if entry.Provider == "vcpkg" {
			hasVcpkg = true
			break
		}
	}
	if !hasVcpkg {
		return "", nil
	}

	v, err := vcpkg.New(e.Path, nil)
	if err != nil {
		return "", fmt.Errorf("resolve vcpkg commit: %w", err)
	}
	return v.VcpkgCommit()
}

// platformKey builds the lock file section key for a given platform target.
func platformKey(p *Platform, platform string) string {
	if platform == "" {
		return p.OS + "." + p.Arch
	}
	return p.OS + "." + p.Arch + "." + platform
}