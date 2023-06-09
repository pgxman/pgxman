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

		overrides[key] = val
	}

	return overrides
}
