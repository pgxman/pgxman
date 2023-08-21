package cmd

import "strings"

func ParseMapFlag(flagSet map[string]string) map[string]any {
	overrides := make(map[string]any)
	for k, v := range flagSet {
		var (
			key string
			val any
		)

		key, val = k, v

		// slice
		valStr := val.(string)
		if strings.HasPrefix(valStr, "[") && strings.HasSuffix(valStr, "]") {
			valStr = strings.TrimPrefix(valStr, "[")
			valStr = strings.TrimSuffix(valStr, "]")

			val = strings.Split(valStr, ",")
		}

		split := strings.Split(key, ".")
		setNestedMap(overrides, split, val)
	}

	return overrides
}

func setNestedMap(m map[string]any, keys []string, val any) {
	if len(keys) == 0 {
		return
	}

	if len(keys) == 1 {
		m[keys[0]] = val
		return
	}

	v, ok := m[keys[0]]
	if !ok {
		v = make(map[string]any)
		m[keys[0]] = v
	}

	setNestedMap(v.(map[string]any), keys[1:], val)
}
