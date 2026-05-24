package vcpkg

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/carbon-os/environment/provider/apt"
	"github.com/carbon-os/environment/provider/brew"
	"github.com/carbon-os/environment/provider/winget"
)

// toolchainPkgs lists the packages vcpkg needs to compile C/C++ ports,
// keyed by host OS. Package names match the native provider's registry.
var toolchainPkgs = map[string][]string{
	"linux":   {"cmake", "ninja-build", "gcc", "g++"},
	"darwin":  {"cmake", "ninja", "gcc"},
	"windows": {"Kitware.CMake", "Ninja-build.Ninja"},
}

// ensureToolchain checks that the required build tools are present on PATH and
// installs any that are missing using the appropriate sibling provider package.
func ensureToolchain(envPath string, log Logger) error {
	pkgs, ok := toolchainPkgs[runtime.GOOS]
	if !ok {
		return nil
	}

	// binaries to probe on PATH (may differ from package name)
	probes := map[string]string{
		"cmake":         "cmake",
		"ninja-build":   "ninja",
		"ninja":         "ninja",
		"gcc":           "gcc",
		"g++":           "g++",
		"Kitware.CMake": "cmake",
		"Ninja-build.Ninja": "ninja",
	}

	var missing []string
	for _, pkg := range pkgs {
		bin := probes[pkg]
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, pkg)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	log.Warn(fmt.Sprintf("vcpkg: installing missing build tools: %v", missing))
	return installToolchain(envPath, missing, log)
}

// installToolchain installs the missing build tools into envPath using the
// host-native provider package — the same providers the rest of the system uses.
func installToolchain(envPath string, pkgs []string, log Logger) error {
	switch runtime.GOOS {
	case "linux":
		a, err := apt.New(envPath, aptLogBridge{log})
		if err != nil {
			return fmt.Errorf("vcpkg: init apt for toolchain: %w", err)
		}
		for _, pkg := range pkgs {
			if err := a.Install(pkg, apt.Params{}); err != nil {
				return fmt.Errorf("vcpkg: toolchain apt install %s: %w", pkg, err)
			}
		}

	case "darwin":
		b, err := brew.New(envPath, brewLogBridge{log})
		if err != nil {
			return fmt.Errorf("vcpkg: init brew for toolchain: %w", err)
		}
		for _, pkg := range pkgs {
			if err := b.Install(pkg, brew.Params{}); err != nil {
				return fmt.Errorf("vcpkg: toolchain brew install %s: %w", pkg, err)
			}
		}

	case "windows":
		w, err := winget.New(envPath, wingetLogBridge{log})
		if err != nil {
			return fmt.Errorf("vcpkg: init winget for toolchain: %w", err)
		}
		for _, pkg := range pkgs {
			if err := w.Install(pkg, winget.Params{}); err != nil {
				return fmt.Errorf("vcpkg: toolchain winget install %s: %w", pkg, err)
			}
		}

	default:
		return fmt.Errorf("vcpkg: unsupported OS %q — install %v manually", runtime.GOOS, pkgs)
	}
	return nil
}

// ── logger bridges ────────────────────────────────────────────────────────────
// Each sibling provider defines its own Logger interface. These thin bridges
// forward events to our Logger without importing the environment package.

type aptLogBridge struct{ l Logger }

func (b aptLogBridge) DepsResolved(_ string, _, _ int)          {}
func (b aptLogBridge) Downloading(n, v string, s int64)         { b.l.Downloading(n, v, s) }
func (b aptLogBridge) DownloadProgress(n string, r, t int64)    { b.l.DownloadProgress(n, r, t) }
func (b aptLogBridge) DownloadDone(n, v string)                 { b.l.DownloadDone(n, v) }
func (b aptLogBridge) Installing(n, v string, pre, dep bool)    { b.l.Installing(n, v, pre, dep) }
func (b aptLogBridge) Installed(n, v string, pre, dep bool)     { b.l.Installed(n, v, pre, dep) }
func (b aptLogBridge) Warn(msg string)                          { b.l.Warn(msg) }

type brewLogBridge struct{ l Logger }

func (b brewLogBridge) DepsResolved(_ string, _, _ int)         {}
func (b brewLogBridge) Downloading(n, v string, s int64)        { b.l.Downloading(n, v, s) }
func (b brewLogBridge) DownloadProgress(n string, r, t int64)   { b.l.DownloadProgress(n, r, t) }
func (b brewLogBridge) DownloadDone(n, v string)                { b.l.DownloadDone(n, v) }
func (b brewLogBridge) Installing(n, v string, pre, dep bool)   { b.l.Installing(n, v, pre, dep) }
func (b brewLogBridge) Installed(n, v string, pre, dep bool)    { b.l.Installed(n, v, pre, dep) }
func (b brewLogBridge) Warn(msg string)                         { b.l.Warn(msg) }

type wingetLogBridge struct{ l Logger }

func (b wingetLogBridge) DepsResolved(_ string, _, _ int)       {}
func (b wingetLogBridge) Downloading(n, v string, s int64)      { b.l.Downloading(n, v, s) }
func (b wingetLogBridge) DownloadProgress(n string, r, t int64) { b.l.DownloadProgress(n, r, t) }
func (b wingetLogBridge) DownloadDone(n, v string)              { b.l.DownloadDone(n, v) }
func (b wingetLogBridge) Installing(n, v string, pre, dep bool) { b.l.Installing(n, v, pre, dep) }
func (b wingetLogBridge) Installed(n, v string, pre, dep bool)  { b.l.Installed(n, v, pre, dep) }
func (b wingetLogBridge) Warn(msg string)                       { b.l.Warn(msg) }