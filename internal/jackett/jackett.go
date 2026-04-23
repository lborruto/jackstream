package jackett

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lborruto/jackstream/internal/config"
	"github.com/lborruto/jackstream/internal/parser"
	"github.com/lborruto/jackstream/internal/tmdb"
)

type Result struct {
	TitleStr   string
	TorrentURL string
	MagnetURI  string
	SizeBytes  int64
	Seeders    int
	InfoHash   string
	TorrentID  string
	Parsed     parser.Parsed
}

// Implement sorter.Sortable + filter.Filterable.
func (r Result) Title() string              { return r.TitleStr }
func (r Result) Size() int64                { return r.SizeBytes }
func (r Result) SeederCount() int           { return r.Seeders }
func (r Result) ParsedTitle() parser.Parsed { return r.Parsed }

type jackettResponse struct {
	Results []struct {
		Title     string `json:"Title"`
		Link      string `json:"Link"`
		MagnetURI string `json:"MagnetUri"`
		Size      int64  `json:"Size"`
		Seeders   int    `json:"Seeders"`
	} `json:"Results"`
}

var (
	httpClient = http.DefaultClient
	btihRe     = regexp.MustCompile(`xt=urn:btih:([a-zA-Z0-9]+)`)
)

func SetHTTPClient(c *http.Client) { httpClient = c }

func requestTimeout() time.Duration {
	ms, _ := strconv.Atoi(os.Getenv("REQUEST_TIMEOUT_MS"))
	if ms <= 0 {
		ms = 8000
	}
	return time.Duration(ms) * time.Millisecond
}

var categoriesByKind = map[string][]string{
	"movie":  {"2000"},
	"series": {"5000", "5070"},
}

func buildQuery(title string, meta tmdb.Meta, kind string) string {
	switch kind {
	case "series":
		if meta.Season > 0 || meta.Episode > 0 {
			return fmt.Sprintf("%s S%02dE%02d", title, meta.Season, meta.Episode)
		}
		return title
	case "movie":
		if meta.Year > 0 {
			return fmt.Sprintf("%s %d", title, meta.Year)
		}
	}
	return title
}

func extractInfoHash(magnet string) string {
	m := btihRe.FindStringSubmatch(magnet)
	if len(m) < 2 {
		return ""
	}
	return strings.ToLower(m[1])
}

func torrentIDOf(src string) string {
	h := sha1.Sum([]byte(src))
	return hex.EncodeToString(h[:])[:12]
}

func searchOneVariant(ctx context.Context, title string, meta tmdb.Meta, kind string, c config.Config) []Result {
	q := buildQuery(title, meta, kind)
	base := strings.TrimRight(c.JackettURL, "/")
	params := url.Values{}
	params.Set("apikey", c.JackettAPIKey)
	params.Set("Query", q)
	for _, cat := range categoriesByKind[kind] {
		params.Add("Category[]", cat)
	}
	u := base + "/api/v2.0/indexers/all/results?" + params.Encode()

	ctx, cancel := context.WithTimeout(ctx, requestTimeout())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[jackett] variant %q failed: %v", title, err)
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		log.Printf("[jackett] variant %q -> HTTP %d", title, res.StatusCode)
		return nil
	}
	var data jackettResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		log.Printf("[jackett] variant %q decode: %v", title, err)
		return nil
	}
	out := make([]Result, 0, len(data.Results))
	for _, r := range data.Results {
		if r.Seeders < c.MinSeeders {
			continue
		}
		ih := extractInfoHash(r.MagnetURI)
		src := r.Link
		if src == "" {
			src = r.MagnetURI
		}
		if src == "" {
			continue
		}
		out = append(out, Result{
			TitleStr:   r.Title,
			TorrentURL: r.Link,
			MagnetURI:  r.MagnetURI,
			SizeBytes:  r.Size,
			Seeders:    r.Seeders,
			InfoHash:   ih,
			TorrentID:  torrentIDOf(src),
			Parsed:     parser.Parse(r.Title),
		})
	}
	return out
}

// Search fans out across titles, dedups by infoHash (fallback to torrent URL), returns combined set.
func Search(ctx context.Context, titles []string, meta tmdb.Meta, kind string, c config.Config) []Result {
	var wg sync.WaitGroup
	results := make([][]Result, len(titles))
	for i, t := range titles {
		i, t := i, t
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = searchOneVariant(ctx, t, meta, kind, c)
		}()
	}
	wg.Wait()

	seen := map[string]struct{}{}
	var merged []Result
	for _, set := range results {
		for _, r := range set {
			key := r.InfoHash
			if key == "" {
				key = "url:" + r.TorrentURL
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, r)
		}
	}
	return merged
}

// Ping returns true if Jackett responds 2xx to a trivial search.
func Ping(ctx context.Context, base, apiKey string) bool {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout())
	defer cancel()
	base = strings.TrimRight(base, "/")
	u := fmt.Sprintf("%s/api/v2.0/indexers/all/results?apikey=%s&Query=test", base, apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	res, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode < 300
}
