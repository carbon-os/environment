package vcpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Logger receives structured events from the vcpkg provider.
// The interface is intentionally identical to the apt/brew/winget loggers so
// the top-level environment.Logger bridges straight across.
type Logger interface {
	Downloading(name, version string, sizeBytes int64)
	DownloadProgress(name string, received, total int64)
	DownloadDone(name, version string)
	Installing(name, version string, isPre, isDep bool)
	Installed(name, version string, isPre, isDep bool)
	Warn(msg string)
}

// Params is the normalised install options passed down from the environment layer.
type Params struct {
	Version      string
	Platform     string // "debian:12", "ubuntu:22.04", "macos", "windows:11", …
	DownloadOnly bool
}

// Vcpkg is the vcpkg provider. One instance is created per environment.
// It owns the vcpkg clone and installed-package tree that live inside the env.
type Vcpkg struct {
	envPath    string // environment root  (~/.env/envs/<name>/)
	vcpkgDir   string // <envPath>/vcpkg/             — the cloned repo + binary
	installDir string // <envPath>/vcpkg_installed/   — built headers/libs/bins
	logger     Logger
}

// New returns a Vcpkg provider rooted at envPath. The vcpkg binary is not
// required to exist yet; ensureBootstrapped() is called lazily on first use.
func New(envPath string, logger Logger) (*Vcpkg, error) {
	if logger == nil {
		logger = noopLogger{}
	}
	return &Vcpkg{
		envPath:    envPath,
		vcpkgDir:   filepath.Join(envPath, "vcpkg"),
		installDir: filepath.Join(envPath, "vcpkg_installed"),
		logger:     logger,
	}, nil
}

// Install builds and installs pkg into the environment.
func (v *Vcpkg) Install(pkg string, params Params) error {
	if err := v.ensureBootstrapped(); err != nil {
		return err
	}
	if err := ensureToolchain(v.envPath, v.logger); err != nil { // fix: was missing v.envPath
		return err
	}

	t, err := ResolveTriplet(params.Platform) // fix: was resolveTriplet (unexported, doesn't exist)
	if err != nil {
		return fmt.Errorf("vcpkg install %s: %w", pkg, err)
	}

	v.logger.Installing(pkg, params.Version, false, false)
	if err := v.runInstall(pkg, params.Version, t, params.DownloadOnly); err != nil {
		return err
	}
	v.logger.Installed(pkg, params.Version, false, false)

	return v.linkBins(t)
}

// Remove uninstalls pkg from the environment.
func (v *Vcpkg) Remove(pkg string) error {
	if err := v.ensureBootstrapped(); err != nil {
		return err
	}

	t, err := ResolveTriplet("") // fix: was resolveTriplet (unexported, doesn't exist)
	if err != nil {
		return fmt.Errorf("vcpkg remove %s: %w", pkg, err)
	}

	return v.runRemove(pkg, t)
}

// Resolve normalises a package name for vcpkg (lowercase, hyphens only).
func (v *Vcpkg) Resolve(pkg string) (string, error) {
	return normaliseName(pkg), nil
}

// binaryPath returns the path to the vcpkg executable inside the clone.
func (v *Vcpkg) binaryPath() string {
	name := "vcpkg"
	if runtime.GOOS == "windows" {
		name = "vcpkg.exe"
	}
	return filepath.Join(v.vcpkgDir, name)
}

// downloadsDir returns the vcpkg downloads cache directory, kept inside the
// environment so different environments don't share potentially stale caches.
func (v *Vcpkg) downloadsDir() string {
	return filepath.Join(v.envPath, "vcpkg_downloads")
}

// ensureDirs makes the required subdirectories under the environment root.
func (v *Vcpkg) ensureDirs() error {
	for _, d := range []string{v.installDir, v.downloadsDir()} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("vcpkg: mkdir %s: %w", d, err)
		}
	}
	return nil
}

// noopLogger discards all events.
type noopLogger struct{}

func (noopLogger) Downloading(_, _ string, _ int64)      {}
func (noopLogger) DownloadProgress(_ string, _, _ int64) {}
func (noopLogger) DownloadDone(_, _ string)              {}
func (noopLogger) Installing(_, _ string, _, _ bool)     {}
func (noopLogger) Installed(_, _ string, _, _ bool)      {}
func (noopLogger) Warn(_ string)                         {}