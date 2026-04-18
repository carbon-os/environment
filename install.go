package environment

import (
	"fmt"

	"github.com/carbon-os/environment/provider/apt"
	"github.com/carbon-os/environment/provider/brew"
	"github.com/carbon-os/environment/provider/winget"
)

// InstallParams controls how a package is installed.
type InstallParams struct {
	Version      string // loose ("13") or pinned ("13.2.1")
	Platform     string // "debian:12", "ubuntu:22.04", "macos", "windows:11"
	Provider     string // explicit override; auto-detected from Platform or host if empty
	DownloadOnly bool   // fetch package but skip exec and post-install steps
}

// Install installs a package into the environment.
func (e *Environment) Install(pkg string, params InstallParams) error {
	// Determine the effective platform label for the log line.
	platformLabel := params.Platform
	if platformLabel == "" {
		platformLabel = e.platform.OS
	}
	e.log().Collecting(pkg, params.Version, platformLabel, e.platform.Arch)

	p, err := e.resolveProvider(params)
	if err != nil {
		return fmt.Errorf("install %s: %w", pkg, err)
	}

	if err := p.Install(pkg, ProviderParams{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	}); err != nil {
		return fmt.Errorf("install %s: %w", pkg, err)
	}

	return e.recordPackage(pkg, params)
}

// Remove removes a package from the environment.
func (e *Environment) Remove(pkg string) error {
	p, err := e.resolveProvider(InstallParams{})
	if err != nil {
		return fmt.Errorf("remove %s: %w", pkg, err)
	}

	if err := p.Remove(pkg); err != nil {
		return fmt.Errorf("remove %s: %w", pkg, err)
	}

	return e.unrecordPackage(pkg)
}

// resolveProvider selects and returns the correct Provider.
// Provider selection priority: explicit params.Provider > inferred from
// params.Platform > host default.
func (e *Environment) resolveProvider(params InstallParams) (Provider, error) {
	name := params.Provider
	if name == "" {
		name = e.platform.DefaultProvider(params.Platform)
	}
	if name == "" {
		return nil, fmt.Errorf("unknown platform %q — supported: debian:11, debian:12, ubuntu:20.04, ubuntu:22.04, ubuntu:24.04, macos, windows:11", params.Platform)
	}

	switch name {
	case "apt":
		a, err := apt.New(e.Path, loggerBridge{e.log()})
		if err != nil {
			return nil, fmt.Errorf("init apt provider: %w", err)
		}
		return &aptAdapter{a}, nil

	case "brew":
		bw, err := brew.New(e.Path, brewLoggerBridge{e.log()})
		if err != nil {
			return nil, fmt.Errorf("init brew provider: %w", err)
		}
		return &brewAdapter{bw}, nil

	case "winget":
		wg, err := winget.New(e.Path, wingetLoggerBridge{e.log()})
		if err != nil {
			return nil, fmt.Errorf("init winget provider: %w", err)
		}
		return &wingetAdapter{wg}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// ── apt ───────────────────────────────────────────────────────────────────────

// aptAdapter bridges *apt.Apt to the top-level Provider interface.
type aptAdapter struct{ a *apt.Apt }

func (x *aptAdapter) Install(pkg string, params ProviderParams) error {
	return x.a.Install(pkg, apt.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}

func (x *aptAdapter) Remove(pkg string) error { return x.a.Remove(pkg) }

func (x *aptAdapter) Resolve(pkg string) (string, error) { return x.a.Resolve(pkg) }

// loggerBridge adapts environment.Logger → apt.Logger so the apt provider
// can fire events without importing the environment package.
type loggerBridge struct{ l Logger }

func (b loggerBridge) DepsResolved(pkg string, pre, deps int) {
	b.l.DepsResolved(pkg, pre, deps)
}
func (b loggerBridge) Downloading(name, version string, size int64) {
	b.l.Downloading(name, version, size)
}
func (b loggerBridge) DownloadProgress(name string, recv, total int64) {
	b.l.DownloadProgress(name, recv, total)
}
func (b loggerBridge) DownloadDone(name, version string) {
	b.l.DownloadDone(name, version)
}
func (b loggerBridge) Installing(name, version string, isPre, isDep bool) {
	b.l.Installing(name, version, isPre, isDep)
}
func (b loggerBridge) Installed(name, version string, isPre, isDep bool) {
	b.l.Installed(name, version, isPre, isDep)
}
func (b loggerBridge) Warn(msg string) { b.l.Warn(msg) }

// ── brew ──────────────────────────────────────────────────────────────────────

// brewAdapter bridges *brew.Brew to the top-level Provider interface.
type brewAdapter struct{ b *brew.Brew }

func (x *brewAdapter) Install(pkg string, params ProviderParams) error {
	return x.b.Install(pkg, brew.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}

func (x *brewAdapter) Remove(pkg string) error { return x.b.Remove(pkg) }

func (x *brewAdapter) Resolve(pkg string) (string, error) { return x.b.Resolve(pkg) }

// brewLoggerBridge adapts environment.Logger → brew.Logger so the brew provider
// can fire events without importing the environment package.
type brewLoggerBridge struct{ l Logger }

func (b brewLoggerBridge) DepsResolved(pkg string, pre, deps int) {
	b.l.DepsResolved(pkg, pre, deps)
}
func (b brewLoggerBridge) Downloading(name, version string, size int64) {
	b.l.Downloading(name, version, size)
}
func (b brewLoggerBridge) DownloadProgress(name string, recv, total int64) {
	b.l.DownloadProgress(name, recv, total)
}
func (b brewLoggerBridge) DownloadDone(name, version string) {
	b.l.DownloadDone(name, version)
}
func (b brewLoggerBridge) Installing(name, version string, isPre, isDep bool) {
	b.l.Installing(name, version, isPre, isDep)
}
func (b brewLoggerBridge) Installed(name, version string, isPre, isDep bool) {
	b.l.Installed(name, version, isPre, isDep)
}
func (b brewLoggerBridge) Warn(msg string) { b.l.Warn(msg) }

// ── winget ────────────────────────────────────────────────────────────────────

// wingetAdapter bridges *winget.Winget to the top-level Provider interface.
type wingetAdapter struct{ w *winget.Winget }

func (x *wingetAdapter) Install(pkg string, params ProviderParams) error {
	return x.w.Install(pkg, winget.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}

func (x *wingetAdapter) Remove(pkg string) error { return x.w.Remove(pkg) }

func (x *wingetAdapter) Resolve(pkg string) (string, error) { return x.w.Resolve(pkg) }

// wingetLoggerBridge adapts environment.Logger → winget.Logger so the winget
// provider can fire events without importing the environment package.
type wingetLoggerBridge struct{ l Logger }

func (b wingetLoggerBridge) DepsResolved(pkg string, pre, deps int) {
	b.l.DepsResolved(pkg, pre, deps)
}
func (b wingetLoggerBridge) Downloading(name, version string, size int64) {
	b.l.Downloading(name, version, size)
}
func (b wingetLoggerBridge) DownloadProgress(name string, recv, total int64) {
	b.l.DownloadProgress(name, recv, total)
}
func (b wingetLoggerBridge) DownloadDone(name, version string) {
	b.l.DownloadDone(name, version)
}
func (b wingetLoggerBridge) Installing(name, version string, isPre, isDep bool) {
	b.l.Installing(name, version, isPre, isDep)
}
func (b wingetLoggerBridge) Installed(name, version string, isPre, isDep bool) {
	b.l.Installed(name, version, isPre, isDep)
}
func (b wingetLoggerBridge) Warn(msg string) { b.l.Warn(msg) }

// ── index ─────────────────────────────────────────────────────────────────────

func (e *Environment) recordPackage(pkg string, params InstallParams) error {
	idx, err := readIndex(e.Path)
	if err != nil {
		return err
	}
	if idx.Packages == nil {
		idx.Packages = make(map[string]PackageEntry)
	}
	idx.Packages[pkg] = PackageEntry{
		Version:  params.Version,
		Platform: params.Platform,
	}
	return writeIndex(e.Path, idx)
}

func (e *Environment) unrecordPackage(pkg string) error {
	idx, err := readIndex(e.Path)
	if err != nil {
		return err
	}
	delete(idx.Packages, pkg)
	return writeIndex(e.Path, idx)
}