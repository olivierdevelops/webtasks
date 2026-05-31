package orchestrator

import (
	"fmt"
	"os"
	"strings"

	"webtasks/internal/domain"
	"webtasks/internal/infra/bundle"
	"webtasks/internal/infra/console"
	"webtasks/internal/infra/yamlreader"
)

// ResolvedSecret describes one secret that was successfully sourced at startup.
// `Source` is "env", "arg", "prompt", or "default".
type ResolvedSecret struct {
	Name      string
	Source    string
	Sensitive bool
}

// LoadSecrets reads `<bundle>/secrets.yaml` (missing is fine), resolves each
// declared secret in source-order, and publishes the value as an env var so
// the templating layer (`{{NAME}}`) can find it. Returns the audit list.
func LoadSecrets(b *bundle.Root, prompter console.Prompter, args []string) ([]ResolvedSecret, error) {
	if !b.Exists("secrets.yaml") {
		return nil, nil
	}
	data, err := b.ReadFile("secrets.yaml")
	if err != nil {
		return nil, err
	}
	var raw struct {
		Secrets []domain.SecretDecl `yaml:"secrets"`
	}
	if err := yamlreader.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	argMap := parseFlagArgs(args)
	out := make([]ResolvedSecret, 0, len(raw.Secrets))
	for _, decl := range raw.Secrets {
		sources := decl.Sources
		if len(sources) == 0 {
			sources = []string{"env", "arg", "prompt"}
		}
		resolved, source, err := resolveOne(decl, sources, argMap, prompter)
		if err != nil {
			return out, err
		}
		if resolved != "" {
			_ = os.Setenv(decl.Name, resolved)
			out = append(out, ResolvedSecret{Name: decl.Name, Source: source, Sensitive: decl.Sensitive})
			continue
		}
		if decl.Required {
			return out, fmt.Errorf("required secret %q not found (tried %v)", decl.Name, sources)
		}
	}
	return out, nil
}

func resolveOne(decl domain.SecretDecl, sources []string, args map[string]string, prompter console.Prompter) (string, string, error) {
	for _, src := range sources {
		switch src {
		case "env":
			if v := os.Getenv(decl.Name); v != "" {
				return v, "env", nil
			}
		case "arg":
			if v, ok := args[decl.Name]; ok && v != "" {
				return v, "arg", nil
			}
		case "prompt":
			// Prompting is only possible with a controlling terminal. Without
			// one we skip this source entirely (for required *and* optional
			// secrets) so a missing required value produces the clean
			// "required secret … not found" error rather than a stdin EOF.
			if !prompter.HasTTY() {
				continue
			}
			value, err := promptFor(decl, prompter)
			if err != nil {
				return "", "", err
			}
			if value != "" {
				return value, "prompt", nil
			}
		}
	}
	if decl.DefaultValue != "" {
		return decl.DefaultValue, "default", nil
	}
	return "", "", nil
}

func promptFor(decl domain.SecretDecl, prompter console.Prompter) (string, error) {
	hint := ""
	if decl.Description != "" {
		hint = " (" + decl.Description + ")"
	}
	prompt := "[webtasks] " + decl.Name + hint + ": "
	if decl.Sensitive {
		return prompter.ReadPassword(prompt)
	}
	return prompter.ReadLine(prompt)
}

// parseFlagArgs collects --key=value entries from os.Args style slices.
func parseFlagArgs(args []string) map[string]string {
	out := map[string]string{}
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			continue
		}
		eq := strings.Index(a, "=")
		if eq < 0 {
			continue
		}
		out[a[2:eq]] = a[eq+1:]
	}
	return out
}
