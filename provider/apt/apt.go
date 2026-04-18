package apt

import (
	"fmt"
	"os"
	"path/filepath"
)

// Params is what the top-level passes down to the apt provider.
// Mirrors ProviderParams — apt has no import dependency on the environment package.
type Params struct {
	Version      string
	Platform     string // "debian:12", "ubuntu:22.04", etc.
	DownloadOnly bool
}

// Apt is the provider for Debian/Ubuntu package installs.
// It is pure file I/O — no apt-get, no system calls, runs on any host OS.
type Apt struct {
	binDir   string // destination for extracted binaries
	cacheDir string // staging area for downloaded .deb files
}

// New returns an Apt provider that installs binaries into binDir.
func New(binDir string) (*Apt, error) {
	cacheDir := filepath.Join(os.TempDir(), "env-apt-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("apt: init cache dir: %w", err)
	}
	return &Apt{binDir: binDir, cacheDir: cacheDir}, nil
}

// Install fetches and unpacks a package from a Debian/Ubuntu mirror.
// Platform selects the distro — defaults to debian:12 if empty.
func (a *Apt) Install(pkg string, params Params) error {
	target := params.Platform
	if target == "" {
		target = "debian:12"
	}

	img, err := resolveImage(target)
	if err != nil {
		return fmt.Errorf("apt install: %w", err)
	}

	index, err := fetchPackageIndex(img)
	if err != nil {
		return fmt.Errorf("apt install: fetch index: %w", err)
	}

	meta, err := findPackage(index, pkg, params.Version)
	if err != nil {
		return fmt.Errorf("apt install: %w", err)
	}

	debPath, err := download(img, meta, a.cacheDir)
	if err != nil {
		return fmt.Errorf("apt install: download: %w", err)
	}

	if params.DownloadOnly {
		return nil
	}

	if err := unpack(debPath, a.binDir); err != nil {
		return fmt.Errorf("apt install: unpack: %w", err)
	}

	return nil
}

// Remove deletes a package's binary from the environment bin dir.
func (a *Apt) Remove(pkg string) error {
	target := filepath.Join(a.binDir, pkg)
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("apt remove %s: %w", pkg, err)
	}
	return nil
}

// Resolve normalises a package name for apt (passthrough for most packages).
func (a *Apt) Resolve(pkg string) (string, error) {
	return pkg, nil
}