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
		{
			url:         "foobar.example.com",
			expectedURL: "//foobar.example.com",
		},
		{
			url:         "foobar.example.com:2376",
			expectedURL: "//foobar.example.com:2376",
		},
		{
			url:         "//foobar.example.com:2376",
			expectedURL: "//foobar.example.com:2376",
		},
		{
			url:         "http://foobar.example.com:2376",
			expectedURL: "http://foobar.example.com:2376",
		},
		{
			url:         "https://foobar.example.com:2376",
			expectedURL: "https://foobar.example.com:2376",
		},
		{
			url:         "https://foobar.example.com:2376/some/path",
			expectedURL: "https://foobar.example.com:2376/some/path",
		},
		{
			url:         "https://foobar.example.com:2376/some/other/path?foo=bar",
			expectedURL: "https://foobar.example.com:2376/some/other/path",
		},
		{
			url: "/foobar.example.com",
			err: errors.New("no hostname in URL"),
		},
		{
			url: "ftp://foobar.example.com:2376",
			err: errors.New("unsupported scheme: ftp"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.url, func(t *testing.T) {
			u, err := Parse(tc.url)

			if tc.err == nil && err != nil {
				t.Fatalf("Error: failed to parse URL %q: %s", tc.url, err)
			}
			if tc.err != nil && err == nil {
				t.Fatalf("Error: expected error %q, got none when parsing URL %q", tc.err, tc.url)
			}
			if tc.err != nil && err.Error() != tc.err.Error() {
				t.Fatalf("Error: expected error %q, got %q when parsing URL %q", tc.err, err, tc.url)
			}
			if u != nil && u.String() != tc.expectedURL {
				t.Errorf("Error: expected URL: %q, but got %q for URL: %q", tc.expectedURL, u.String(), tc.url)
			}
		})
	}
}
