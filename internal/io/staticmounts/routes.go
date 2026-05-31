// Package staticmounts serves user-configured directories at user-configured
// URL prefixes. The mount table is read at startup from the bundle's
// static-mounts.yaml — no URLs are hardcoded in Go.
package staticmounts

import (
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"webtasks/internal/domain"
)

func Register(r chi.Router, mounts []domain.StaticMount) {
	for _, m := range mounts {
		m := m
		prefix := normalisePrefix(m.Prefix)
		if m.List {
			r.Get(prefix, listing(prefix, m))
		}
		if m.Serve {
			r.Get(prefix+"/*", serve(m))
		}
	}
}

func normalisePrefix(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}

type entry struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"`
}

func listing(prefix string, m domain.StaticMount) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		dir, _ := filepath.Abs(m.Dir)
		entries := walkEntries(dir, prefix, m.Recursive)
		body := map[string]any{
			"ok":      true,
			"mount":   prefix,
			"dir":     dir,
			"count":   len(entries),
			"entries": entries,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

func walkEntries(root, prefix string, recursive bool) []entry {
	out := []entry{}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return out
	}
	walk := func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		urlPath := prefix + "/" + filepath.ToSlash(rel)
		out = append(out, entry{
			Name:  filepath.ToSlash(rel),
			URL:   urlPath,
			Size:  info.Size(),
			Mtime: info.ModTime().UnixMilli(),
		})
		return nil
	}
	if recursive {
		_ = filepath.WalkDir(root, walk)
	} else {
		ds, _ := os.ReadDir(root)
		for _, d := range ds {
			_ = walk(filepath.Join(root, d.Name()), d, nil)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func serve(m domain.StaticMount) http.HandlerFunc {
	dirAbs, _ := filepath.Abs(m.Dir)
	return func(w http.ResponseWriter, req *http.Request) {
		rel := chi.URLParam(req, "*")
		if rel == "" {
			http.NotFound(w, req)
			return
		}
		clean := filepath.Clean(rel)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		target := filepath.Join(dirAbs, clean)
		if !strings.HasPrefix(target+string(os.PathSeparator), dirAbs+string(os.PathSeparator)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		info, err := os.Stat(target)
		if err != nil || info.IsDir() {
			http.NotFound(w, req)
			return
		}
		if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(target))); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		http.ServeFile(w, req, target)
	}
}
