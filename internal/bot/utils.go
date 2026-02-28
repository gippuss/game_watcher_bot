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

// splitNames разбивает payload на названия. Запятая — разделитель.
// Названия с запятыми внутри можно брать в кавычки: "Игра, часть вторая", Другая игра.
func splitNames(payload string) []string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}
	var out []string
	for len(payload) > 0 {
		payload = strings.TrimLeft(payload, " \t")
		if payload == "" {
			break
		}
		if payload[0] == '"' {
			end := strings.Index(payload[1:], `"`)
			if end == -1 {
				out = append(out, strings.TrimSpace(payload[1:]))
				break
			}
			out = append(out, strings.TrimSpace(payload[1:1+end]))
			payload = payload[1+end+1:]
			payload = strings.TrimLeft(payload, " \t")
			if len(payload) > 0 && payload[0] == ',' {
				payload = payload[1:]
			}
			continue
		}
		idx := strings.Index(payload, ",")
		if idx == -1 {
			name := strings.TrimSpace(payload)
			if name != "" {
				out = append(out, name)
			}
			break
		}
		name := strings.TrimSpace(payload[:idx])
		if name != "" {
			out = append(out, name)
		}
		payload = payload[idx+1:]
	}
	return out
}
