package winget

// Logger receives structured events from the winget provider.
// Mirrors the apt.Logger interface so the same bridge in install.go works.
// A nil Logger silences all output — every call site guards with a nil check.
type Logger interface {
	// DepsResolved fires after dependency resolution. winget has no dep graph,
	// so preDeps and deps will always be 0.
	DepsResolved(pkg string, preDeps, deps int)
	// Downloading fires when the installer download begins.
	// sizeBytes is Content-Length, or -1 if unknown.
	Downloading(name, version string, sizeBytes int64)
	// DownloadProgress fires repeatedly as bytes arrive.
	DownloadProgress(name string, received, total int64)
	// DownloadDone fires when the download completes.
	DownloadDone(name, version string)
	// Installing fires just before the installer is unpacked.
	Installing(name, version string, isPre, isDep bool)
	// Installed fires after the package is successfully placed.
	Installed(name, version string, isPre, isDep bool)
	// Warn fires for non-fatal advisories.
	Warn(msg string)
}