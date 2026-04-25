package parser

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Byrutgame: название в <div class="hname"><h1>...</h1>, версия в <div class="subhnamever js-ver">...</div>
var (
	byrutNameRE    = regexp.MustCompile(`class="hname"[^>]*>\s*<h1>([^<]*)</h1>`)
	byrutVersionRE = regexp.MustCompile(`class="subhnamever js-ver"[^>]*>([^<]*)</div>`)
)

// Паттерны для прочих сайтов (semver и похожие).
var versionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:version|версия|v)\s*[:\s=]\s*["']?(\d+\.\d+(?:\.\d+)?(?:[-.\w]*)?)["']?`),
	regexp.MustCompile(`(\d+\.\d+\.\d+(?:[-.\w]*)?)`),
	regexp.MustCompile(`(\d+\.\d+(?:\.\d+)?)`),
}

// GameInfo — название и актуальная версия игры со страницы.
type GameInfo struct {
	Name          string // из <div class="hname"><h1>...</h1>
	LatestVersion string // из <div class="subhnamever js-ver">...</div>
}

// FetchGameInfo загружает страницу по URL и извлекает название и версию (один запрос).
// Для byrutgame.org — из .hname h1 и div#version.
func FetchGameInfo(ctx context.Context, rawURL string) (*GameInfo, error) {
	html, err := fetchPage(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	if html == "" {
		return nil, errors.New("страница пуста")
	}
	u, _ := url.Parse(rawURL)
	if u != nil && isByrutHost(u.Hostname()) {
		name := extractByrutName(html)
		ver := extractByrutVersion(html)
		return &GameInfo{Name: name, LatestVersion: ver}, nil
	}
	ver := extractVersionFallback(html)
	return &GameInfo{LatestVersion: ver}, nil
}

// LatestVersion загружает страницу по URL и извлекает только версию (для /check).
func LatestVersion(ctx context.Context, rawURL string) (string, error) {
	info, err := FetchGameInfo(ctx, rawURL)
	if err != nil || info == nil {
		return "", err
	}
	return info.LatestVersion, nil
}

func fetchPage(ctx context.Context, rawURL string) (string, error) {
	return fetchPageDirect(ctx, rawURL)
}

func isByrutHost(host string) bool {
	return host == "byrutgame.org" || strings.HasSuffix(host, ".byrutgame.org")
}

func fetchPageDirect(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GameWatcherBot/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	const maxBody = 512 * 1024
	if resp.ContentLength > maxBody {
		return "", nil
	}
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 8*1024)
	for total := 0; total < maxBody; {
		n, err := resp.Body.Read(tmp)
		total += n
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return string(buf), nil
}

func extractByrutName(html string) string {
	matches := byrutNameRE.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	name := strings.TrimSpace(matches[1])
	if name == "" || len(name) > 500 {
		return ""
	}
	return name
}

func extractByrutVersion(html string) string {
	matches := byrutVersionRE.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	v := strings.TrimSpace(matches[1])
	if v == "" || len(v) > 500 {
		return ""
	}
	return v
}

func extractVersionFallback(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	for _, re := range versionPatterns {
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 {
			v := strings.TrimSpace(matches[1])
			if v != "" && len(v) < 50 {
				return v
			}
		}
	}
	return ""
}
