package registryurl

import (
	"errors"
	"testing"
)

// TestHelperParseURL verifies that a // "scheme" is added to URLs,
// and that invalid URLs produce an error.
func TestHelperParseURL(t *testing.T) {
	tests := []struct {
		url         string
		expectedURL string
		err         error
	}{
		{url: "foobar.docker.io", expectedURL: "//foobar.docker.io"},
		{url: "foobar.docker.io:2376", expectedURL: "//foobar.docker.io:2376"},
		{url: "//foobar.docker.io:2376", expectedURL: "//foobar.docker.io:2376"},
		{url: "http://foobar.docker.io:2376", expectedURL: "http://foobar.docker.io:2376"},
		{url: "https://foobar.docker.io:2376", expectedURL: "https://foobar.docker.io:2376"},
		{url: "https://foobar.docker.io:2376/some/path", expectedURL: "https://foobar.docker.io:2376/some/path"},
		{url: "https://foobar.docker.io:2376/some/other/path?foo=bar", expectedURL: "https://foobar.docker.io:2376/some/other/path"},
		{url: "/foobar.docker.io", err: errors.New("no hostname in URL")},
		{url: "ftp://foobar.docker.io:2376", err: errors.New("unsupported scheme: ftp")},
	}

	for _, te := range tests {
		u, err := Parse(te.url)

		if te.err == nil && err != nil {
			t.Errorf("Error: failed to parse URL %q: %s", te.url, err)
			continue
		}
		if te.err != nil && err == nil {
			t.Errorf("Error: expected error %q, got none when parsing URL %q", te.err, te.url)
			continue
		}
		if te.err != nil && err.Error() != te.err.Error() {
			t.Errorf("Error: expected error %q, got %q when parsing URL %q", te.err, err, te.url)
			continue
		}
		if u != nil && u.String() != te.expectedURL {
			t.Errorf("Error: expected URL: %q, but got %q for URL: %q", te.expectedURL, u.String(), te.url)
		}
	}
}
