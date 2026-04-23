package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lborruto/jackstream/internal/bt"
	"github.com/lborruto/jackstream/internal/config"
	"github.com/lborruto/jackstream/internal/filter"
	"github.com/lborruto/jackstream/internal/jackett"
	"github.com/lborruto/jackstream/internal/parser"
	"github.com/lborruto/jackstream/internal/sorter"
	"github.com/lborruto/jackstream/internal/store"
	"github.com/lborruto/jackstream/internal/tmdb"
)

// configure.html is copied from ../../public/configure.html so that
// //go:embed can consume it (embed does not follow symlinks). Regenerate with:
//
//go:generate sh -c "cp ../../public/configure.html configure.html"
//
//go:embed configure.html
var configureHTML []byte

type Manifest struct {
	ID            string          `json:"id"`
	Version       string          `json:"version"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Resources     []string        `json:"resources"`
	Types         []string        `json:"types"`
	IDPrefixes    []string        `json:"idPrefixes"`
	BehaviorHints map[string]bool `json:"behaviorHints"`
	Catalogs      []any           `json:"catalogs"`
}

func baseManifest() Manifest {
	return Manifest{
		ID:          "community.jackstream",
		Version:     "1.1.0",
		Name:        "jackstream",
		Description: "Stream torrents from your Jackett instance directly in Stremio. Self-hosted, no debrid required.",
		Resources:   []string{"stream"},
		Types:       []string{"movie", "series"},
		IDPrefixes:  []string{"tt"},
		BehaviorHints: map[string]bool{
			"configurable":          true,
			"configurationRequired": true,
		},
		Catalogs: []any{},
	}
}

func configuredManifest() Manifest {
	m := baseManifest()
	m.BehaviorHints = map[string]bool{
		"configurable":          true,
		"configurationRequired": false,
	}
	return m
}

type Server struct {
	bt        *bt.Client
	maxConcur int
}

func New(c *bt.Client) *Server { return &Server{bt: c, maxConcur: 3} }

func (s *Server) SetMaxConcurrent(n int) {
	if n > 0 {
		s.maxConcur = n
	}
}

func (s *Server) MaxConcurrent() int { return s.maxConcur }

func (s *Server) BuildHandler() http.Handler {
	// The "stream list" route `/{config}/stream/{type}/{id...}` and the
	// "stream file" route `/stream/{config}/{torrentID}/{fileIdx}` overlap
	// when a base64 config literally equals the word "stream", which Go's
	// ServeMux rejects as ambiguous. We split them into two muxes keyed on
	// the first path segment: "/stream/..." is file-playback; everything
	// else (including the configured-manifest routes) uses the addon mux.
	addon := http.NewServeMux()
	addon.HandleFunc("GET /health", s.handleHealth)
	addon.HandleFunc("GET /manifest.json", s.handleManifestRoot)
	addon.HandleFunc("GET /{config}/manifest.json", s.handleManifestConfigured)
	addon.HandleFunc("GET /{config}/stream/{type}/{id...}", s.handleStreamList)
	addon.HandleFunc("GET /api/test-jackett", s.handleTestJackett)
	addon.HandleFunc("GET /api/test-tmdb", s.handleTestTMDB)
	addon.HandleFunc("GET /configure", s.handleConfigure)
	addon.HandleFunc("GET /{config}/configure", s.handleConfigure)
	addon.HandleFunc("GET /", s.handleRoot)

	playback := http.NewServeMux()
	playback.HandleFunc("GET /stream/{config}/{torrentID}/{fileIdx}", s.handleStreamFile)

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/stream/") {
			playback.ServeHTTP(w, r)
			return
		}
		addon.ServeHTTP(w, r)
	})
	return corsMiddleware(root)
}

func corsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleManifestRoot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, baseManifest())
}

func (s *Server) handleManifestConfigured(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("config")
	decoded, err := config.Decode(raw)
	if err == nil {
		_, err = config.Validate(decoded)
	}
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid_config", "detail": err.Error()})
		return
	}
	writeJSON(w, 200, configuredManifest())
}

func (s *Server) handleStreamList(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("config")
	kind := r.PathValue("type")
	idPath := strings.TrimSuffix(r.PathValue("id"), ".json")

	cfg, err := config.Decode(raw)
	if err == nil {
		cfg, err = config.Validate(cfg)
	}
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid_config", "detail": err.Error()})
		return
	}
	s.SetMaxConcurrent(cfg.MaxConcurrentTorrents)

	meta, err := tmdb.Resolve(r.Context(), idPath, cfg.TMDBAPIKey)
	if err != nil {
		writeJSON(w, 200, map[string]any{"streams": []any{}})
		return
	}
	titles := tmdb.TitleVariants(meta)
	results := jackett.Search(r.Context(), titles, meta, kind, cfg)

	filterable := make([]filter.Filterable, 0, len(results))
	for _, rr := range results {
		filterable = append(filterable, rr)
	}
	filtered := filter.Filter(filterable, cfg)

	sortable := make([]sorter.Sortable, 0, len(filtered))
	for _, f := range filtered {
		sortable = append(sortable, f.(jackett.Result))
	}
	sorted := sorter.Sort(sortable, cfg.PreferredLanguage)
	if len(sorted) > cfg.MaxResults {
		sorted = sorted[:cfg.MaxResults]
	}

	base := strings.TrimRight(cfg.AddonPublicURL, "/")
	streams := make([]map[string]any, 0, len(sorted))
	for _, sv := range sorted {
		rr := sv.(jackett.Result)
		store.Set(rr.TorrentID, store.TorrentMeta{
			TorrentURL:  rr.TorrentURL,
			MagnetURI:   rr.MagnetURI,
			Size:        rr.SizeBytes,
			Seeders:     rr.Seeders,
			InfoHash:    rr.InfoHash,
			ParsedTitle: rr.Parsed,
		})
		streams = append(streams, map[string]any{
			"name":        streamName(rr.Parsed),
			"description": streamDescription(rr),
			"url":         fmt.Sprintf("%s/stream/%s/%s/0", base, raw, rr.TorrentID),
			"behaviorHints": map[string]any{
				"notWebReady": true,
				"bingeGroup":  "jackstream",
			},
		})
	}

	writeJSON(w, 200, map[string]any{"streams": streams})
}

func streamName(p parser.Parsed) string {
	parts := []string{}
	if p.Quality != "" {
		parts = append(parts, p.Quality)
	}
	if p.HDR != "" {
		parts = append(parts, p.HDR)
	}
	q := strings.Join(parts, " ")
	if q == "" {
		q = "—"
	}
	return "🎬 Jackett\n" + q
}

func streamDescription(r jackett.Result) string {
	size := formatSize(r.SizeBytes)
	src := ""
	if r.Parsed.Source != "" {
		src = " • 🔵 " + r.Parsed.Source
	}
	aud := ""
	if r.Parsed.Audio != "" {
		aud = " • " + r.Parsed.Audio
	}
	return fmt.Sprintf("%s\n👥 %d seeds • 💾 %s%s%s", r.TitleStr, r.Seeders, size, src, aud)
}

func formatSize(bytes int64) string {
	if bytes <= 0 {
		return "?"
	}
	gb := float64(bytes) / (1 << 30)
	if gb >= 1 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	mb := float64(bytes) / (1 << 20)
	return fmt.Sprintf("%.0f MB", mb)
}

func (s *Server) handleTestJackett(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("url")
	key := r.URL.Query().Get("key")
	if u == "" || key == "" {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "missing url or key"})
		return
	}
	ok := jackett.Ping(r.Context(), u, key)
	writeJSON(w, 200, map[string]bool{"ok": ok})
}

func (s *Server) handleTestTMDB(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "missing key"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"https://api.themoviedb.org/3/configuration?api_key="+key, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, 200, map[string]bool{"ok": false})
		return
	}
	defer res.Body.Close()
	writeJSON(w, 200, map[string]bool{"ok": res.StatusCode < 300})
}

func (s *Server) handleConfigure(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(configureHTML)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/configure", http.StatusFound)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
