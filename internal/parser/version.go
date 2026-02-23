package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Byrutgame: название в <div class="hname"><h1>...</h1>, версия в <div id="version">...</div>
var (
	byrutNameRE    = regexp.MustCompile(`class="hname"[^>]*>\s*<h1>([^<]*)</h1>`)
	byrutVersionRE = regexp.MustCompile(`id="version"[^>]*>([^<]*)</div>`)
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
	LatestVersion string // из <div id="version">...</div>
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
	u, _ := url.Parse(rawURL)
	if u != nil && isByrutHost(u.Hostname()) {
		baseURL := getFlareSolverrBaseURL()
		if baseURL != "" {
			return fetchPageViaFlareSolverr(ctx, baseURL, rawURL)
		}
	}
	return fetchPageDirect(ctx, rawURL)
}

func isByrutHost(host string) bool {
	return host == "byrutgame.org" || strings.HasSuffix(host, ".byrutgame.org")
}

// flaresolverrURLs — список URL FlareSolverr из FLARESOLVERR_URL (через запятую), round-robin счётчик.
var (
	flaresolverrURLs     []string
	flaresolverrNext     atomic.Uint32
	flaresolverrURLsOnce sync.Once
)

func getFlareSolverrBaseURL() string {
	flaresolverrURLsOnce.Do(func() {
		s := os.Getenv("FLARESOLVERR_URL")
		if s == "" {
			return
		}
		for _, u := range strings.Split(s, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				flaresolverrURLs = append(flaresolverrURLs, u)
			}
		}
	})
	if len(flaresolverrURLs) == 0 {
		return ""
	}
	if len(flaresolverrURLs) == 1 {
		return flaresolverrURLs[0]
	}
	n := flaresolverrNext.Add(1)
	return flaresolverrURLs[int(n)%len(flaresolverrURLs)]
}

// fetchPageViaFlareSolverr запрашивает страницу через FlareSolverr (обходит Cloudflare).
func fetchPageViaFlareSolverr(ctx context.Context, flaresolverrBaseURL, targetURL string) (string, error) {
	body := struct {
		Cmd        string `json:"cmd"`
		URL        string `json:"url"`
		MaxTimeout int    `json:"maxTimeout"`
	}{
		Cmd:        "request.get",
		URL:        targetURL,
		MaxTimeout: 60000,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, flaresolverrBaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	var result struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Solution *struct {
			Response string `json:"response"`
		} `json:"solution"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Status != "ok" || result.Solution == nil {
		return "", nil
	}
	html := result.Solution.Response
	if len(html) > 512*1024 {
		html = html[:512*1024]
	}
	return html, nil
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
