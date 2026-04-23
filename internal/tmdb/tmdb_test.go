package tmdb

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func respOK(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func resp404() *http.Response {
	return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("{}"))}
}

func TestParseStremioIDMovie(t *testing.T) {
	id, err := ParseStremioID("tt0111161")
	if err != nil {
		t.Fatal(err)
	}
	if id.IMDB != "tt0111161" || id.Season != 0 || id.Episode != 0 {
		t.Errorf("movie: %#v", id)
	}
}

func TestParseStremioIDSeries(t *testing.T) {
	id, err := ParseStremioID("tt0903747:1:3")
	if err != nil {
		t.Fatal(err)
	}
	if id.IMDB != "tt0903747" || id.Season != 1 || id.Episode != 3 {
		t.Errorf("series: %#v", id)
	}
}

func TestParseStremioIDRejectsGarbage(t *testing.T) {
	if _, err := ParseStremioID("not-an-id"); err == nil {
		t.Error("expected error")
	}
}

func TestResolveMovie(t *testing.T) {
	ClearCache()
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		if strings.Contains(r.URL.Path, "/find/tt0111161") {
			return respOK(`{"movie_results":[{"id":1,"title":"The Shawshank Redemption","original_title":"The Shawshank Redemption","release_date":"1994-09-23"}],"tv_results":[]}`)
		}
		return resp404()
	})
	SetHTTPClient(&http.Client{Transport: rt})
	defer SetHTTPClient(http.DefaultClient)

	meta, err := Resolve(context.Background(), "tt0111161", "key")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if meta.Type != "movie" || meta.Year != 1994 || meta.Title != "The Shawshank Redemption" {
		t.Errorf("movie meta: %#v", meta)
	}
}

func TestResolveCachesRepeated(t *testing.T) {
	ClearCache()
	calls := 0
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		calls++
		return respOK(`{"movie_results":[{"id":1,"title":"X","original_title":"X","release_date":"2020-01-01"}],"tv_results":[]}`)
	})
	SetHTTPClient(&http.Client{Transport: rt})
	defer SetHTTPClient(http.DefaultClient)

	_, _ = Resolve(context.Background(), "tt0111161", "k")
	_, _ = Resolve(context.Background(), "tt0111161", "k")
	if calls != 1 {
		t.Errorf("expected 1 TMDB call, got %d", calls)
	}
}

func TestResolveSeriesEpisode(t *testing.T) {
	ClearCache()
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/find/tt0903747"):
			return respOK(`{"movie_results":[],"tv_results":[{"id":1396,"name":"Breaking Bad","original_name":"Breaking Bad","first_air_date":"2008-01-20"}]}`)
		case strings.Contains(p, "/tv/1396/season/1/episode/3"):
			return respOK(`{"name":"...And the Bag's in the River"}`)
		}
		return resp404()
	})
	SetHTTPClient(&http.Client{Transport: rt})
	defer SetHTTPClient(http.DefaultClient)

	meta, err := Resolve(context.Background(), "tt0903747:1:3", "k")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if meta.Type != "series" || meta.Season != 1 || meta.Episode != 3 {
		t.Errorf("series meta: %#v", meta)
	}
	if !strings.Contains(meta.EpisodeTitle, "Bag") {
		t.Errorf("episode title missing: %q", meta.EpisodeTitle)
	}
}
