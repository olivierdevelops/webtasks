package orchestrator

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"webtasks/internal/infra/capytranspile"
)

// BundleDir packages a source directory into a runnable bundle zip. `.webtask`
// recipes are transpiled to YAML on the way in (so the server, which loads
// YAML, can run them); every other file is copied verbatim.
//
//	webtasks bundle <src-dir> [out.zip]
//
// The resulting zip is what WEBTASKS_BUNDLE points at:
//
//	WEBTASKS_BUNDLE=out.zip webtasks
func BundleDir(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: webtasks bundle <src-dir> [out.zip]")
	}
	srcDir := args[0]
	info, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("read source %q: %w", srcDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source must be a directory: %s", srcDir)
	}

	out := strings.TrimRight(filepath.Base(filepath.Clean(srcDir)), string(filepath.Separator)) + ".zip"
	if len(args) > 1 {
		out = args[1]
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}

	zf, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create %q: %w", out, err)
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	defer zw.Close()

	var recipes, copied int
	walkErr := filepath.Walk(srcDir, func(p string, fi os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if fi.IsDir() {
			return nil
		}
		abs, _ := filepath.Abs(p)
		if abs == absOut {
			// Don't swallow the output zip if it lives inside srcDir.
			return nil
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if capytranspile.IsRecipe(p) {
			yamlBytes, terr := capytranspile.ToYAML(p)
			if terr != nil {
				return terr
			}
			dst := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".yaml"
			if err := writeZipEntry(zw, dst, yamlBytes); err != nil {
				return err
			}
			recipes++
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := writeZipEntry(zw, rel, data); err != nil {
			return err
		}
		copied++
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	if err := zw.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[webtasks] bundled %d recipe(s) + %d file(s) -> %s\n", recipes, copied, out)
	fmt.Fprintf(os.Stderr, "[webtasks] run it with: WEBTASKS_BUNDLE=%s webtasks\n", out)
	return nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, strings.NewReader(string(data)))
	return err
}
