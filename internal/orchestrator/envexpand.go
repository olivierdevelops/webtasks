package orchestrator

import (
	"os"
	"regexp"
)

// expandEnv resolves `${NAME}` and `${NAME:-default}` placeholders in a string.
// Used for static-mounts.yaml `dir:` values so deployments can point them at
// the right host paths without editing the YAML.
var envExpandRE = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?::-([^}]*))?\}`)

func expandEnv(s string) string {
	if s == "" {
		return s
	}
	return envExpandRE.ReplaceAllStringFunc(s, func(match string) string {
		m := envExpandRE.FindStringSubmatch(match)
		if v := os.Getenv(m[1]); v != "" {
			return v
		}
		if len(m) > 2 {
			return m[2]
		}
		return ""
	})
}
