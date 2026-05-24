package environment

import (
	"fmt"

	"github.com/carbon-os/environment/provider/apt"
	"github.com/carbon-os/environment/provider/brew"
	"github.com/carbon-os/environment/provider/vcpkg"
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
// Priority: explicit params.Provider > inferred from params.Platform > host default.
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

	case "vcpkg":
		vg, err := vcpkg.New(e.Path, vcpkgLoggerBridge{e.log()})
		if err != nil {
			return nil, fmt.Errorf("init vcpkg provider: %w", err)
		}
		return &vcpkgAdapter{vg}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// ── apt ───────────────────────────────────────────────────────────────────────

type aptAdapter struct{ a *apt.Apt }

func (x *aptAdapter) Install(pkg string, params ProviderParams) error {
	return x.a.Install(pkg, apt.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}
func (x *aptAdapter) Remove(pkg string) error              { return x.a.Remove(pkg) }
func (x *aptAdapter) Resolve(pkg string) (string, error)   { return x.a.Resolve(pkg) }

type loggerBridge struct{ l Logger }

func (b loggerBridge) DepsResolved(pkg string, pre, deps int)      { b.l.DepsResolved(pkg, pre, deps) }
func (b loggerBridge) Downloading(n, v string, s int64)            { b.l.Downloading(n, v, s) }
func (b loggerBridge) DownloadProgress(n string, r, t int64)       { b.l.DownloadProgress(n, r, t) }
func (b loggerBridge) DownloadDone(n, v string)                    { b.l.DownloadDone(n, v) }
func (b loggerBridge) Installing(n, v string, pre, dep bool)       { b.l.Installing(n, v, pre, dep) }
func (b loggerBridge) Installed(n, v string, pre, dep bool)        { b.l.Installed(n, v, pre, dep) }
func (b loggerBridge) Warn(msg string)                             { b.l.Warn(msg) }

// ── brew ──────────────────────────────────────────────────────────────────────

type brewAdapter struct{ b *brew.Brew }

func (x *brewAdapter) Install(pkg string, params ProviderParams) error {
	return x.b.Install(pkg, brew.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}
func (x *brewAdapter) Remove(pkg string) error              { return x.b.Remove(pkg) }
func (x *brewAdapter) Resolve(pkg string) (string, error)   { return x.b.Resolve(pkg) }

type brewLoggerBridge struct{ l Logger }

func (b brewLoggerBridge) DepsResolved(pkg string, pre, deps int)  { b.l.DepsResolved(pkg, pre, deps) }
func (b brewLoggerBridge) Downloading(n, v string, s int64)        { b.l.Downloading(n, v, s) }
func (b brewLoggerBridge) DownloadProgress(n string, r, t int64)   { b.l.DownloadProgress(n, r, t) }
func (b brewLoggerBridge) DownloadDone(n, v string)                { b.l.DownloadDone(n, v) }
func (b brewLoggerBridge) Installing(n, v string, pre, dep bool)   { b.l.Installing(n, v, pre, dep) }
func (b brewLoggerBridge) Installed(n, v string, pre, dep bool)    { b.l.Installed(n, v, pre, dep) }
func (b brewLoggerBridge) Warn(msg string)                         { b.l.Warn(msg) }

// ── winget ────────────────────────────────────────────────────────────────────

type wingetAdapter struct{ w *winget.Winget }

func (x *wingetAdapter) Install(pkg string, params ProviderParams) error {
	return x.w.Install(pkg, winget.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}
func (x *wingetAdapter) Remove(pkg string) error              { return x.w.Remove(pkg) }
func (x *wingetAdapter) Resolve(pkg string) (string, error)   { return x.w.Resolve(pkg) }

type wingetLoggerBridge struct{ l Logger }

func (b wingetLoggerBridge) DepsResolved(pkg string, pre, deps int)  { b.l.DepsResolved(pkg, pre, deps) }
func (b wingetLoggerBridge) Downloading(n, v string, s int64)        { b.l.Downloading(n, v, s) }
func (b wingetLoggerBridge) DownloadProgress(n string, r, t int64)   { b.l.DownloadProgress(n, r, t) }
func (b wingetLoggerBridge) DownloadDone(n, v string)                { b.l.DownloadDone(n, v) }
func (b wingetLoggerBridge) Installing(n, v string, pre, dep bool)   { b.l.Installing(n, v, pre, dep) }
func (b wingetLoggerBridge) Installed(n, v string, pre, dep bool)    { b.l.Installed(n, v, pre, dep) }
func (b wingetLoggerBridge) Warn(msg string)                         { b.l.Warn(msg) }

// ── vcpkg ─────────────────────────────────────────────────────────────────────

type vcpkgAdapter struct{ v *vcpkg.Vcpkg }

func (x *vcpkgAdapter) Install(pkg string, params ProviderParams) error {
	return x.v.Install(pkg, vcpkg.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}
func (x *vcpkgAdapter) Remove(pkg string) error              { return x.v.Remove(pkg) }
func (x *vcpkgAdapter) Resolve(pkg string) (string, error)   { return x.v.Resolve(pkg) }

// vcpkgLoggerBridge adapts environment.Logger → vcpkg.Logger.
// vcpkg.Logger has no DepsResolved (vcpkg handles dep resolution internally),
// so that event is simply dropped here.
type vcpkgLoggerBridge struct{ l Logger }

func (b vcpkgLoggerBridge) Downloading(n, v string, s int64)        { b.l.Downloading(n, v, s) }
func (b vcpkgLoggerBridge) DownloadProgress(n string, r, t int64)   { b.l.DownloadProgress(n, r, t) }
func (b vcpkgLoggerBridge) DownloadDone(n, v string)                { b.l.DownloadDone(n, v) }
func (b vcpkgLoggerBridge) Installing(n, v string, pre, dep bool)   { b.l.Installing(n, v, pre, dep) }
func (b vcpkgLoggerBridge) Installed(n, v string, pre, dep bool)    { b.l.Installed(n, v, pre, dep) }
func (b vcpkgLoggerBridge) Warn(msg string)                         { b.l.Warn(msg) }

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
		Provider: params.Provider, // persisted so lock/sync round-trips correctly
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