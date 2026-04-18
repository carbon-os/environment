package apt

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blakesmith/ar"
	"github.com/ulikunitz/xz"
)

// unpack extracts /usr/bin binaries from a .deb into binDir.
//
// .deb structure (ar archive):
//   debian-binary   — format version
//   control.tar.*   — package metadata (skipped)
//   data.tar.*      — actual files (we extract /usr/bin/* from here)
func unpack(debPath, binDir string) error {
	f, err := os.Open(debPath)
	if err != nil {
		return fmt.Errorf("unpack: open: %w", err)
	}
	defer f.Close()

	reader := ar.NewReader(f)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unpack: read ar: %w", err)
		}

		if !strings.HasPrefix(header.Name, "data.tar") {
			continue
		}

		return extractDataTar(reader, header.Name, binDir)
	}

	return fmt.Errorf("unpack: data.tar not found in %s", debPath)
}

// extractDataTar decompresses and extracts /usr/bin entries from data.tar.*.
func extractDataTar(r io.Reader, name, binDir string) error {
	tr, err := decompressedTar(r, name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("extract: mkdir binDir: %w", err)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("extract: read tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Only extract files under /usr/bin
		normalized := strings.TrimPrefix(header.Name, "./")
		if !strings.HasPrefix(normalized, "usr/bin/") {
			continue
		}

		base := filepath.Base(normalized)
		destPath := filepath.Join(binDir, base)

		if err := writeFile(destPath, tr, header.Mode); err != nil {
			return fmt.Errorf("extract: write %s: %w", base, err)
		}
	}

	return nil
}

// decompressedTar wraps r in the correct decompressor based on the data.tar filename.
func decompressedTar(r io.Reader, name string) (*tar.Reader, error) {
	switch {
	case strings.HasSuffix(name, ".gz"):
		gz, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("decompress gz: %w", err)
		}
		return tar.NewReader(gz), nil

	case strings.HasSuffix(name, ".xz"):
		xzr, err := xz.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("decompress xz: %w", err)
		}
		return tar.NewReader(xzr), nil

	case strings.HasSuffix(name, ".bz2"):
		return tar.NewReader(bzip2.NewReader(r)), nil

	case strings.HasSuffix(name, ".zst"):
		return nil, fmt.Errorf("zstd not yet supported")

	default:
		return nil, fmt.Errorf("unknown data.tar compression: %s", name)
	}
}

func writeFile(path string, r io.Reader, mode int64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}