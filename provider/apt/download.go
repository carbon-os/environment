package apt

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// download fetches a .deb from the mirror into the cache dir and returns its path.
func download(img image, meta packageMeta, cacheDir string) (string, error) {
	url := packageURL(img, meta.Filename)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("download: mkdir: %w", err)
	}

	debName := filepath.Base(meta.Filename)
	destPath := filepath.Join(cacheDir, debName)

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("download: create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("download: write: %w", err)
	}

	return destPath, nil
}