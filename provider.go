package environment

// ProviderParams is the normalised set of install options passed to any provider.
// Providers only receive this — they have no knowledge of InstallParams or the
// top-level environment API.
type ProviderParams struct {
	Version      string
	Platform     string // "debian:12", "ubuntu:22.04", "macos", "windows:11"
	DownloadOnly bool   // fetch only, skip exec and post-install steps
}

// Provider is the interface all package providers must implement.
// Each provider owns all logic for its target platform — routing, downloading,
// unpacking. The top level only calls through this interface.
type Provider interface {
	Install(pkg string, params ProviderParams) error
	Remove(pkg string) error
	Resolve(pkg string) (string, error) // normalise pkg name for this provider
}