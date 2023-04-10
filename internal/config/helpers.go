package config

import "strings"

func cleanList(in []string) []string {
	out := make([]string, len(in))

	for i, v := range in {
		out[i] = strings.TrimSpace(v)
	}

	return out
}
