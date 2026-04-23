package config

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

var stdEncoder = base64.StdEncoding

func sample() Config {
	return Config{
		JackettURL:     "http://192.168.1.10:9117",
		JackettAPIKey:  "abc",
		TMDBAPIKey:     "def",
		AddonPublicURL: "https://addon.example.com",
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	encoded, err := Encode(sample())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.JackettAPIKey != "abc" || got.TMDBAPIKey != "def" {
		t.Errorf("round-trip mismatch: %#v", got)
	}
}

func TestEncodeIsURLSafe(t *testing.T) {
	encoded, err := Encode(sample())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if regexp.MustCompile(`[+/=]`).MatchString(encoded) {
		t.Errorf("encoded form contains URL-unsafe chars: %s", encoded)
	}
}

func TestDecodeAcceptsStandardBase64(t *testing.T) {
	s := `{"jackettUrl":"http://x","jackettApiKey":"a","tmdbApiKey":"b","addonPublicUrl":"http://y"}`
	stdB64 := stdEncoder.EncodeToString([]byte(s))
	got, err := Decode(stdB64)
	if err != nil {
		t.Fatalf("Decode(standard): %v", err)
	}
	if got.JackettAPIKey != "a" {
		t.Errorf("expected JackettAPIKey=a, got %q", got.JackettAPIKey)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	_, err := Decode("!!!not-b64!!!")
	if err == nil {
		t.Error("expected error on garbage input, got nil")
	}
}

func TestValidatePasses(t *testing.T) {
	if _, err := Validate(sample()); err != nil {
		t.Errorf("Validate(valid): %v", err)
	}
}

func TestValidateRequiredFields(t *testing.T) {
	for _, field := range []string{"JackettURL", "JackettAPIKey", "TMDBAPIKey", "AddonPublicURL"} {
		c := sample()
		switch field {
		case "JackettURL":
			c.JackettURL = ""
		case "JackettAPIKey":
			c.JackettAPIKey = ""
		case "TMDBAPIKey":
			c.TMDBAPIKey = ""
		case "AddonPublicURL":
			c.AddonPublicURL = ""
		}
		_, err := Validate(c)
		if err == nil {
			t.Errorf("%s empty: expected error, got nil", field)
			continue
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("%s empty: error must mention field name, got %q", field, err.Error())
		}
	}
}

func TestValidateRejectsBadURL(t *testing.T) {
	c := sample()
	c.JackettURL = "not a url"
	if _, err := Validate(c); err == nil {
		t.Error("expected URL error, got nil")
	}
}

func TestValidateDefaults(t *testing.T) {
	c, err := Validate(sample())
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if c.MaxResults != 10 || c.MinSeeders != 1 || c.MaxConcurrentTorrents != 3 {
		t.Errorf("defaults not applied: %#v", c)
	}
}

func TestValidateKeepsExplicitNumbers(t *testing.T) {
	c := sample()
	c.MaxResults = 25
	c.MinSeeders = 5
	c.MaxConcurrentTorrents = 7
	out, err := Validate(c)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if out.MaxResults != 25 || out.MinSeeders != 5 || out.MaxConcurrentTorrents != 7 {
		t.Errorf("explicit values overridden: %#v", out)
	}
}
