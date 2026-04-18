package environment

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform holds the detected host OS and architecture.
type Platform struct {
	OS   string // darwin, linux, windows
	Arch string // amd64, arm64
}

func detectPlatform() (*Platform, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	switch os {
	case "darwin", "linux", "windows":
	default:
		return nil, fmt.Errorf("unsupported OS: %s", os)
	}

	switch arch {
	case "amd64", "arm64":
	default:
		return nil, fmt.Errorf("unsupported arch: %s", arch)
	}

	return &Platform{OS: os, Arch: arch}, nil
}

// DefaultProvider returns the provider for a given platform target string.
// If platform is empty, falls back to the host OS default.
func (p *Platform) DefaultProvider(platform string) string {
	if platform != "" {
		return providerForPlatform(platform)
	}
	return p.hostDefaultProvider()
}

// providerForPlatform maps a --platform value to a provider name.
func providerForPlatform(platform string) string {
	switch {
	case strings.HasPrefix(platform, "debian:"),
		strings.HasPrefix(platform, "ubuntu:"):
		return "apt"
	case platform == "macos":
		return "brew"
	case strings.HasPrefix(platform, "windows"):
		return "winget"
	default:
		return "apt"
	}
}

// hostDefaultProvider returns the best provider for the current host OS.
func (p *Platform) hostDefaultProvider() string {
	switch p.OS {
	case "darwin":
		return "brew"
	case "linux":
		return "apt"
	case "windows":
		return "winget"
	default:
		return "apt"
	}
}