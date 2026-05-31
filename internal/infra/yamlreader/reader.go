// Package yamlreader is a thin wrapper over gopkg.in/yaml.v3 that returns
// generic `map[string]any` trees the same way SnakeYAML does on the Java side.
package yamlreader

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ReadMap unmarshals YAML bytes into a generic map.
func ReadMap(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	m, ok := normaliseKeys(raw).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("yaml root is not a map: %T", raw)
	}
	return m, nil
}

// Unmarshal binds YAML bytes into a strongly typed struct.
func Unmarshal(data []byte, out any) error { return yaml.Unmarshal(data, out) }

// normaliseKeys converts `map[interface{}]interface{}` (yaml.v2 quirk) into
// `map[string]any` recursively. yaml.v3 already gives us string keys so this
// is a no-op in practice, but kept for safety.
func normaliseKeys(v any) any {
	switch t := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprintf("%v", k)] = normaliseKeys(val)
		}
		return out
	case map[string]any:
		for k, val := range t {
			t[k] = normaliseKeys(val)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = normaliseKeys(x)
		}
		return t
	default:
		return v
	}
}
