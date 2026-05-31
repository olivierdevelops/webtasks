// Package orchestrator owns composition: it imports concrete types from every
// other layer and wires them together. This is the only package that's
// allowed to do so (per VHCO).
package orchestrator

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config carries everything the App needs to boot. Populate via FromEnv() to
// honour the standard WEBTASKS_* variables; tests construct it directly.
type Config struct {
	Host        string
	Port        int
	BundlePath  string // dir or .zip — resolved by infra/bundle.Open
	DownloadDir string
	Headless    bool
}

func FromEnv() Config {
	cwd, _ := os.Getwd()
	return Config{
		Host:        getenv("WEBTASKS_HOST", "127.0.0.1"),
		Port:        atoi(getenv("WEBTASKS_PORT", "8765"), 8765),
		BundlePath:  getenv("WEBTASKS_BUNDLE", filepath.Join(cwd, "bundle-example")),
		DownloadDir: getenv("WEBTASKS_DOWNLOADS_DIR", filepath.Join(cwd, "build", "downloads")),
		Headless:    getenv("WEBTASKS_HEADLESS", "false") == "true",
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoi(s string, d int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return d
}
