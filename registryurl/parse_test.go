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

	for _, tc := range tests {
		u, err := Parse(tc.url)

		if tc.err == nil && err != nil {
			t.Errorf("Error: failed to parse URL %q: %s", tc.url, err)
			continue
		}
		if tc.err != nil && err == nil {
			t.Errorf("Error: expected error %q, got none when parsing URL %q", tc.err, tc.url)
			continue
		}
		if tc.err != nil && err.Error() != tc.err.Error() {
			t.Errorf("Error: expected error %q, got %q when parsing URL %q", tc.err, err, tc.url)
			continue
		}
		if u != nil && u.String() != tc.expectedURL {
			t.Errorf("Error: expected URL: %q, but got %q for URL: %q", tc.expectedURL, u.String(), tc.url)
		}
	}
}
