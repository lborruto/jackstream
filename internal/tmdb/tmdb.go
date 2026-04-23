package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lborruto/jackstream/internal/cache"
)

type Meta struct {
	Type         string
	Title        string
	TitleFr      string
	TitleEn      string
	Year         int
	Season       int
	Episode      int
	EpisodeTitle string
}

type ID struct {
	IMDB    string
	Season  int
	Episode int
}

const tmdbBase = "https://api.themoviedb.org/3"

var (
	httpClient = http.DefaultClient
	findCache  = cache.New[findResponse](time.Now)
	epCache    = cache.New[string](time.Now)
)

type findResponse struct {
	MovieResults []struct {
		ID            int    `json:"id"`
		Title         string `json:"title"`
		OriginalTitle string `json:"original_title"`
		ReleaseDate   string `json:"release_date"`
	} `json:"movie_results"`
	TVResults []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		OriginalName string `json:"original_name"`
		FirstAirDate string `json:"first_air_date"`
	} `json:"tv_results"`
}

type epResponse struct {
	Name string `json:"name"`
}

func SetHTTPClient(c *http.Client) { httpClient = c }

func ClearCache() {
	findCache.Clear()
	epCache.Clear()
}

func ttl() time.Duration {
	mins, _ := strconv.Atoi(os.Getenv("CACHE_TTL_MINUTES"))
	if mins <= 0 {
		mins = 1440
	}
	return time.Duration(mins) * time.Minute
}

func requestTimeout() time.Duration {
	ms, _ := strconv.Atoi(os.Getenv("REQUEST_TIMEOUT_MS"))
	if ms <= 0 {
		ms = 8000
	}
	return time.Duration(ms) * time.Millisecond
}

var idRe = regexp.MustCompile(`^tt\d+$`)

func ParseStremioID(raw string) (ID, error) {
	if raw == "" {
		return ID{}, errors.New("empty id")
	}
	parts := strings.Split(raw, ":")
	imdb := parts[0]
	if !idRe.MatchString(imdb) {
		return ID{}, fmt.Errorf("invalid imdb id: %s", raw)
	}
	switch len(parts) {
	case 1:
		return ID{IMDB: imdb}, nil
	case 3:
		s, err1 := strconv.Atoi(parts[1])
		e, err2 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil {
			return ID{}, fmt.Errorf("invalid series id: %s", raw)
		}
		return ID{IMDB: imdb, Season: s, Episode: e}, nil
	default:
		return ID{}, fmt.Errorf("invalid id: %s", raw)
	}
}

func fetchJSON(ctx context.Context, url string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("TMDB %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func findByIMDB(ctx context.Context, imdb, apiKey string) (findResponse, error) {
	key := "find:" + imdb
	if v, ok := findCache.Get(key); ok {
		return v, nil
	}
	var out findResponse
	url := fmt.Sprintf("%s/find/%s?external_source=imdb_id&api_key=%s", tmdbBase, imdb, apiKey)
	if err := fetchJSON(ctx, url, &out); err != nil {
		return findResponse{}, err
	}
	findCache.Set(key, out, ttl())
	return out, nil
}

func fetchEpisodeTitle(ctx context.Context, tvID, season, episode int, apiKey string) string {
	key := fmt.Sprintf("ep:%d:%d:%d", tvID, season, episode)
	if v, ok := epCache.Get(key); ok {
		return v
	}
	var out epResponse
	url := fmt.Sprintf("%s/tv/%d/season/%d/episode/%d?api_key=%s", tmdbBase, tvID, season, episode, apiKey)
	if err := fetchJSON(ctx, url, &out); err != nil {
		return ""
	}
	epCache.Set(key, out.Name, ttl())
	return out.Name
}

func year(date string) int {
	if len(date) < 4 {
		return 0
	}
	y, _ := strconv.Atoi(date[:4])
	return y
}

func Resolve(ctx context.Context, raw, apiKey string) (Meta, error) {
	id, err := ParseStremioID(raw)
	if err != nil {
		return Meta{}, err
	}
	find, err := findByIMDB(ctx, id.IMDB, apiKey)
	if err != nil {
		return Meta{}, err
	}

	if id.Season != 0 && len(find.TVResults) > 0 {
		tv := find.TVResults[0]
		return Meta{
			Type:         "series",
			Title:        tv.Name,
			TitleFr:      tv.Name,
			TitleEn:      tv.OriginalName,
			Year:         year(tv.FirstAirDate),
			Season:       id.Season,
			Episode:      id.Episode,
			EpisodeTitle: fetchEpisodeTitle(ctx, tv.ID, id.Season, id.Episode, apiKey),
		}, nil
	}

	if len(find.MovieResults) > 0 {
		m := find.MovieResults[0]
		return Meta{
			Type:    "movie",
			Title:   m.Title,
			TitleFr: m.Title,
			TitleEn: m.OriginalTitle,
			Year:    year(m.ReleaseDate),
		}, nil
	}

	if len(find.TVResults) > 0 {
		tv := find.TVResults[0]
		return Meta{
			Type:    "series",
			Title:   tv.Name,
			TitleFr: tv.Name,
			TitleEn: tv.OriginalName,
			Year:    year(tv.FirstAirDate),
		}, nil
	}

	return Meta{}, fmt.Errorf("TMDB: no result for %s", id.IMDB)
}

func TitleVariants(m Meta) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range []string{m.Title, m.TitleFr, m.TitleEn} {
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
