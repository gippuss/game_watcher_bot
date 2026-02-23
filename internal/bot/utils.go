package bot

import (
	"strings"
)

func splitNames(payload string) []string {
	var out []string
	for _, s := range strings.Split(payload, ",") {
		name := strings.TrimSpace(s)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}
