package orchestrator

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:inittemplates
var initTemplates embed.FS

// InitProject scaffolds a starter bundle (sample recipes, a JS module, pool
// config, and a COMMANDS.md reference) into a new directory.
//
//	webtasks init [dir] [--force]
//
// Default dir is "webtasks-starter". Refuses to write into a non-empty
// directory unless --force is given.
func InitProject(args []string) error {
	dir := "webtasks-starter"
	force := false
	seenDir := false
	for _, a := range args {
		switch {
		case a == "--force" || a == "-f":
			force = true
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %q (usage: webtasks init [dir] [--force])", a)
		default:
			if seenDir {
				return fmt.Errorf("usage: webtasks init [dir] [--force]")
			}
			dir, seenDir = a, true
		}
	}

	if info, err := os.Stat(dir); err == nil && info.IsDir() && !force {
		if entries, _ := os.ReadDir(dir); len(entries) > 0 {
			return fmt.Errorf("%q already exists and is not empty — use --force to scaffold into it", dir)
		}
	}

	const root = "inittemplates"
	var count int
	err := fs.WalkDir(initTemplates, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := initTemplates.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		count++
		fmt.Fprintf(os.Stderr, "  + %s\n", dst)
		return nil
	})
	if err != nil {
		return fmt.Errorf("scaffold into %q: %w", dir, err)
	}

	fmt.Fprintf(os.Stderr, "\n[webtasks] created %d files in %s/\n\n", count, dir)
	fmt.Fprintf(os.Stderr, "Next steps:\n")
	fmt.Fprintf(os.Stderr, "  cd %s\n", dir)
	fmt.Fprintf(os.Stderr, "  webtasks run tasks/hello.webtask     # run one recipe\n")
	fmt.Fprintf(os.Stderr, "  WEBTASKS_BUNDLE=. webtasks           # serve them all over HTTP\n")
	fmt.Fprintf(os.Stderr, "  cat COMMANDS.md                      # the full command reference\n")
	return nil
}
