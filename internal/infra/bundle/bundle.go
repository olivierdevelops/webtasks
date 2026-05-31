// Package bundle abstracts a config root that may be a directory on disk OR a
// zip file read in-place via archive/zip (no extraction).
package bundle

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Root opens a bundle from `pathOrZip`. Returns an fs.FS rooted at the bundle.
// When the path points at a `.zip` / `.jar`, the archive is opened and read
// in-place; no temp files are written.
type Root struct {
	src     string
	fsys    fs.FS
	zipper  *zip.ReadCloser
	rootDir string
}

func Open(pathOrZip string) (*Root, error) {
	if pathOrZip == "" {
		return nil, errors.New("bundle path is empty")
	}
	abs, err := filepath.Abs(pathOrZip)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return &Root{src: abs, fsys: os.DirFS(abs), rootDir: abs}, nil
	}
	ext := strings.ToLower(filepath.Ext(abs))
	if ext != ".zip" && ext != ".jar" {
		return nil, fmt.Errorf("bundle must be a directory or .zip/.jar: %s", abs)
	}
	rc, err := zip.OpenReader(abs)
	if err != nil {
		return nil, err
	}
	return &Root{src: abs, fsys: rc, zipper: rc}, nil
}

func (r *Root) Close() {
	if r.zipper != nil {
		_ = r.zipper.Close()
	}
}

func (r *Root) Source() string { return r.src }
func (r *Root) Kind() string {
	if r.zipper != nil {
		return "zip"
	}
	return "dir"
}

// FS returns the underlying file system. Use fs.ReadFile / fs.WalkDir against it.
func (r *Root) FS() fs.FS { return r.fsys }

// ReadFile reads `name` (a slash-separated path relative to the bundle root)
// and returns its bytes. Returns ErrNotExist if missing.
func (r *Root) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(r.fsys, path.Clean(name))
}

// Exists reports whether `name` is in the bundle.
func (r *Root) Exists(name string) bool {
	_, err := fs.Stat(r.fsys, path.Clean(name))
	return err == nil
}

// WalkYAML invokes fn for every `*.yaml` / `*.yml` file under `dir`. The path
// passed to fn is bundle-relative and uses forward slashes.
func (r *Root) WalkYAML(dir string, fn func(relPath string, content []byte) error) error {
	if !r.Exists(dir) {
		return nil
	}
	return fs.WalkDir(r.fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(p)
		if !(strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")) {
			return nil
		}
		f, err := r.fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		return fn(p, content)
	})
}
