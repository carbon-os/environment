package environment

// Logger receives structured install events from the environment package.
// All methods must be safe for concurrent calls.
// The library itself never prints anything — attach a Logger to get output.
type Logger interface {
	// Collecting fires once at the top of Install, before any network I/O.
	Collecting(pkg, version, platform, arch string)
	// DepsResolved fires after the full transitive dependency graph is known.
	DepsResolved(pkg string, preDeps, deps int)
	// Downloading fires when a .deb download begins.
	// sizeBytes is Content-Length from the mirror, or -1 if unknown.
	Downloading(pkg, version string, sizeBytes int64)
	// DownloadProgress fires repeatedly as bytes arrive.
	// total mirrors the value passed to Downloading.
	DownloadProgress(pkg string, received, total int64)
	// DownloadDone fires when a download completes successfully.
	DownloadDone(pkg, version string)
	// Installing fires just before a .deb is unpacked into the environment.
	Installing(pkg, version string, isPre, isDep bool)
	// Installed fires after a package is successfully unpacked.
	Installed(pkg, version string, isPre, isDep bool)
	// Warn fires for non-fatal advisories (e.g. a skipped dep alternative).
	Warn(msg string)
}

// NoopLogger discards all events. It is the default when no Logger is attached.
type NoopLogger struct{}

func (NoopLogger) Collecting(_, _, _, _ string)         {}
func (NoopLogger) DepsResolved(_ string, _, _ int)       {}
func (NoopLogger) Downloading(_, _ string, _ int64)      {}
func (NoopLogger) DownloadProgress(_ string, _, _ int64) {}
func (NoopLogger) DownloadDone(_, _ string)              {}
func (NoopLogger) Installing(_, _ string, _, _ bool)     {}
func (NoopLogger) Installed(_, _ string, _, _ bool)      {}
func (NoopLogger) Warn(_ string)                         {}