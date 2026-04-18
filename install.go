package environment

import (
	"fmt"

	"github.com/carbon-os/environment/provider/apt"
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
// Provider selection priority: explicit params.Provider > inferred from params.Platform > host default.
func (e *Environment) resolveProvider(params InstallParams) (Provider, error) {
	name := params.Provider
	if name == "" {
		name = e.platform.DefaultProvider(params.Platform)
	}

	switch name {
	case "apt":
		a, err := apt.New(e.BinPath())
		if err != nil {
			return nil, fmt.Errorf("init apt provider: %w", err)
		}
		return &aptAdapter{a}, nil
	// brew, winget: coming
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// aptAdapter bridges *apt.Apt to the top-level Provider interface,
// mapping ProviderParams to apt.Params. This keeps apt free of any
// import dependency on the environment package.
type aptAdapter struct{ a *apt.Apt }

func (x *aptAdapter) Install(pkg string, params ProviderParams) error {
	return x.a.Install(pkg, apt.Params{
		Version:      params.Version,
		Platform:     params.Platform,
		DownloadOnly: params.DownloadOnly,
	})
}

func (x *aptAdapter) Remove(pkg string) error {
	return x.a.Remove(pkg)
}

func (x *aptAdapter) Resolve(pkg string) (string, error) {
	return x.a.Resolve(pkg)
}

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