// Package capytranspile turns a .webtask recipe (Capy source) into the YAML
// TaskDef the engine consumes. It shells out to the `capy` CLI, feeding it a
// grammar that is embedded in the binary so end users never manage grammar
// files themselves.
//
// This is a raw infra adapter: it knows nothing about the rest of the system
// and only moves bytes (a path in, YAML out).
package capytranspile

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//go:generate cp ../../../capy/webtasks.capy webtasks.capy

//go:embed webtasks.capy
var grammar []byte

var (
	grammarOnce sync.Once
	grammarPath string
	grammarErr  error
)

// IsRecipe reports whether path looks like a .webtask Capy source file.
func IsRecipe(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".webtask")
}

// Available reports whether the `capy` transpiler is on PATH.
func Available() bool {
	_, err := exec.LookPath("capy")
	return err == nil
}

// ToYAML transpiles a .webtask file to YAML TaskDef bytes. It requires the
// `capy` CLI to be installed (https://github.com/olivierdevelops/capy).
func ToYAML(recipePath string) ([]byte, error) {
	if _, err := exec.LookPath("capy"); err != nil {
		return nil, fmt.Errorf("capy CLI not found on PATH — install it to use .webtask files " +
			"(see https://github.com/olivierdevelops/capy), or pass a .yaml task instead")
	}
	gp, err := embeddedGrammar()
	if err != nil {
		return nil, err
	}
	out, err := exec.Command("capy", "run", gp, recipePath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %v\n%s", recipePath, err, out)
	}
	return out, nil
}

// embeddedGrammar writes the embedded grammar to a stable temp file once per
// process and returns its path.
func embeddedGrammar() (string, error) {
	grammarOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "webtasks-grammar")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			grammarErr = err
			return
		}
		p := filepath.Join(dir, "webtasks.capy")
		if err := os.WriteFile(p, grammar, 0o644); err != nil {
			grammarErr = err
			return
		}
		grammarPath = p
	})
	return grammarPath, grammarErr
}
