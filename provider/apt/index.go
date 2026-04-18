package apt

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// packageMeta holds the resolved metadata for a single package from the Packages index.
type packageMeta struct {
	Name     string
	Version  string
	Filename string // relative pool path e.g. pool/main/g/gcc/gcc_13.2.0_amd64.deb
	SHA256   string
}

// fetchPackageIndex downloads and parses the gzipped Packages index for an image.
func fetchPackageIndex(img image) (map[string]packageMeta, error) {
	url := packageIndexURL(img)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch index %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch index %s: status %d", url, resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decompress index: %w", err)
	}
	defer gz.Close()

	return parsePackageIndex(gz)
}

// parsePackageIndex parses the Debian Packages stanza format into a name-keyed map.
func parsePackageIndex(r io.Reader) (map[string]packageMeta, error) {
	packages := make(map[string]packageMeta)
	scanner := bufio.NewScanner(r)

	// Debian Packages files can have very long lines (base64 checksums).
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var current packageMeta

	flush := func() {
		if current.Name != "" {
			packages[current.Name] = current
			current = packageMeta{}
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			flush()
			continue
		}

		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		switch key {
		case "Package":
			current.Name = value
		case "Version":
			current.Version = value
		case "Filename":
			current.Filename = value
		case "SHA256":
			current.SHA256 = value
		}
	}

	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	return packages, nil
}

// findPackage looks up a package by name with an optional version prefix match.
func findPackage(index map[string]packageMeta, pkg, version string) (packageMeta, error) {
	meta, ok := index[pkg]
	if !ok {
		return packageMeta{}, fmt.Errorf("package %q not found in index", pkg)
	}

	if version != "" && !strings.HasPrefix(meta.Version, version) {
		return packageMeta{}, fmt.Errorf(
			"package %q: requested version %q, available %q",
			pkg, version, meta.Version,
		)
	}

	return meta, nil
}