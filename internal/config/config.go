package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	JackettURL            string   `json:"jackettUrl"`
	JackettAPIKey         string   `json:"jackettApiKey"`
	TMDBAPIKey            string   `json:"tmdbApiKey"`
	AddonPublicURL        string   `json:"addonPublicUrl"`
	MaxResults            int      `json:"maxResults,omitempty"`
	MinSeeders            int      `json:"minSeeders,omitempty"`
	MaxConcurrentTorrents int      `json:"maxConcurrentTorrents,omitempty"`
	PreferredLanguage     string   `json:"preferredLanguage,omitempty"`
	MaxQuality            string   `json:"maxQuality,omitempty"`
	MinQuality            string   `json:"minQuality,omitempty"`
	MaxSizeGb             float64  `json:"maxSizeGb,omitempty"`
	MinSizeMb             float64  `json:"minSizeMb,omitempty"`
	BlacklistKeywords     []string `json:"blacklistKeywords,omitempty"`
}

var urlEnc = base64.URLEncoding.WithPadding(base64.NoPadding)

func Encode(c Config) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return urlEnc.EncodeToString(b), nil
}

func Decode(s string) (Config, error) {
	var raw []byte
	var err error
	if strings.ContainsAny(s, "-_") {
		raw, err = urlEnc.DecodeString(s)
	} else {
		raw, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(s)
		}
	}
	if err != nil {
		return Config{}, fmt.Errorf("decode base64: %w", err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("decode json: %w", err)
	}
	return c, nil
}

var validLangs = map[string]struct{}{
	"FRENCH": {}, "MULTI": {}, "VOSTFR": {}, "ENG": {},
}

var validQualities = map[string]struct{}{
	"4K": {}, "1080p": {}, "720p": {}, "480p": {},
}

func Validate(c Config) (Config, error) {
	if c.JackettURL == "" {
		return Config{}, errors.New("config.JackettURL is required")
	}
	if c.JackettAPIKey == "" {
		return Config{}, errors.New("config.JackettAPIKey is required")
	}
	if c.TMDBAPIKey == "" {
		return Config{}, errors.New("config.TMDBAPIKey is required")
	}
	if c.AddonPublicURL == "" {
		return Config{}, errors.New("config.AddonPublicURL is required")
	}
	if _, err := url.ParseRequestURI(c.JackettURL); err != nil {
		return Config{}, fmt.Errorf("config.JackettURL is not a valid URL: %w", err)
	}
	if _, err := url.ParseRequestURI(c.AddonPublicURL); err != nil {
		return Config{}, fmt.Errorf("config.AddonPublicURL is not a valid URL: %w", err)
	}
	if c.MaxResults <= 0 {
		c.MaxResults = 10
	}
	if c.MinSeeders <= 0 {
		c.MinSeeders = 1
	}
	if c.MaxConcurrentTorrents <= 0 {
		c.MaxConcurrentTorrents = 3
	}
	if _, ok := validLangs[c.PreferredLanguage]; !ok {
		c.PreferredLanguage = ""
	}
	if _, ok := validQualities[c.MaxQuality]; !ok {
		c.MaxQuality = ""
	}
	if _, ok := validQualities[c.MinQuality]; !ok {
		c.MinQuality = ""
	}
	if c.MaxSizeGb < 0 {
		c.MaxSizeGb = 0
	}
	if c.MinSizeMb < 0 {
		c.MinSizeMb = 0
	}
	return c, nil
}
