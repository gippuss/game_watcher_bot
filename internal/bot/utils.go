package bot

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultConc = 3
	envConcKey  = "CONCURRENCY"
)

func concurrencyFromEnv() int {
	s := os.Getenv(envConcKey)
	if s == "" {
		return defaultConc
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultConc
	}
	if n > 20 {
		return 20
	}
	return n
}

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
