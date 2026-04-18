package apt

// Logger receives structured events from the apt provider.
// A nil Logger silences all output — every call site guards with a nil check.
type Logger interface {
	// DepsResolved fires after the full transitive graph is computed.
	DepsResolved(pkg string, preDeps, deps int)
	// Downloading fires when a .deb download starts.
	// sizeBytes is Content-Length, or -1 if the server did not send one.
	Downloading(name, version string, sizeBytes int64)
	// DownloadProgress fires repeatedly as bytes arrive.
	DownloadProgress(name string, received, total int64)
	// DownloadDone fires when a download completes.
	DownloadDone(name, version string)
	// Installing fires just before a .deb is unpacked.
	Installing(name, version string, isPre, isDep bool)
	// Installed fires after a package is unpacked successfully.
	Installed(name, version string, isPre, isDep bool)
	// Warn fires for non-fatal advisories.
	Warn(msg string)
}