//go:build windows

package wincred

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestWinCredHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
		Username:  "foobar",
		Secret:    "foobarbaz",
	}
	creds1 := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v2",
		Username:  "foobarbaz",
		Secret:    "foobar",
	}

	helper := Wincred{}

	// check for and remove remaining credentials from previous fail tests
	oldauths, err := helper.List()
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range oldauths {
		if strings.Compare(k, creds.ServerURL) == 0 && strings.Compare(v, creds.Username) == 0 {
			if err := helper.Delete(creds.ServerURL); err != nil {
				t.Fatal(err)
			}
		} else if strings.Compare(k, creds1.ServerURL) == 0 && strings.Compare(v, creds1.Username) == 0 {
			if err := helper.Delete(creds1.ServerURL); err != nil {
				t.Fatal(err)
			}
		}
	}

	// recount for credentials
	oldauths, err = helper.List()
	if err != nil {
		t.Fatal(err)
	}

	if err := helper.Add(creds); err != nil {
		t.Fatal(err)
	}

	username, secret, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Fatal(err)
	}

	if username != "foobar" {
		t.Fatalf("expected %s, got %s\n", "foobar", username)
	}

	if secret != "foobarbaz" {
		t.Fatalf("expected %s, got %s\n", "foobarbaz", secret)
	}

	auths, err := helper.List()
	if err != nil || len(auths)-len(oldauths) != 1 {
		t.Fatal(err)
	}

	helper.Add(creds1)
	defer helper.Delete(creds1.ServerURL)
	newauths, err := helper.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(newauths)-len(auths) != 1 {
		if err == nil {
			t.Fatalf("Error: len(newauths): %d, len(auths): %d", len(newauths), len(auths))
		}
		t.Fatalf("Error: len(newauths): %d, len(auths): %d\n Error= %v", len(newauths), len(auths), err)
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}
}

// TestWinCredHelperRetrieveAliases verifies that secrets can be accessed
// through variations on the URL
func TestWinCredHelperRetrieveAliases(t *testing.T) {
	tests := []struct {
		doc      string
		storeURL string
		readURL  string
	}{
		{
			doc:      "stored with port, retrieved without",
			storeURL: "https://foobar.docker.io:2376",
			readURL:  "https://foobar.docker.io",
		},
		{
			doc:      "stored as https, retrieved without scheme",
			storeURL: "https://foobar.docker.io",
			readURL:  "foobar.docker.io",
		},
		{
			doc:      "stored with path, retrieved without",
			storeURL: "https://foobar.docker.io/one/two",
			readURL:  "https://foobar.docker.io",
		},
	}

	helper := Wincred{}
	t.Cleanup(func() {
		for _, tc := range tests {
			if err := helper.Delete(tc.storeURL); err != nil && !credentials.IsErrCredentialsNotFound(err) {
				t.Errorf("cleanup: failed to delete '%s': %v", tc.storeURL, err)
			}
		}
	})

	// Clean store before testing.
	for _, tc := range tests {
		if err := helper.Delete(tc.storeURL); err != nil && !credentials.IsErrCredentialsNotFound(err) {
			t.Errorf("prepare: failed to delete '%s': %v", tc.storeURL, err)
		}
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			c := &credentials.Credentials{ServerURL: tc.storeURL, Username: "hello", Secret: "world"}
			if err := helper.Add(c); err != nil {
				t.Fatalf("Error: failed to store secret for URL %q: %s", tc.storeURL, err)
			}
			if _, _, err := helper.Get(tc.readURL); err != nil {
				t.Errorf("Error: failed to read secret for URL %q using %q", tc.storeURL, tc.readURL)
			}
			if err := helper.Delete(tc.storeURL); err != nil {
				t.Error(err)
			}
		})
	}
}

// TestWinCredHelperRetrieveStrict verifies that only matching secrets are
// returned.
func TestWinCredHelperRetrieveStrict(t *testing.T) {
	tests := []struct {
		doc      string
		storeURL string
		readURL  string
	}{
		{
			doc:      "stored as https, retrieved using http",
			storeURL: "https://foobar.docker.io:2376",
			readURL:  "http://foobar.docker.io:2376",
		},
		{
			doc:      "stored as http, retrieved using https",
			storeURL: "http://foobar.docker.io:2376",
			readURL:  "https://foobar.docker.io:2376",
		},
		{
			// stored as http, retrieved without a scheme specified (hence, using the default https://)
			doc:      "stored as http, retrieved without scheme",
			storeURL: "http://foobar.docker.io",
			readURL:  "foobar.docker.io:5678",
		},
		{
			doc:      "non-matching ports",
			storeURL: "https://foobar.docker.io:1234",
			readURL:  "https://foobar.docker.io:5678",
		},
		// TODO: is this desired behavior? The other way round does work
		// {
		// 	doc:      "non-matching ports (stored without port)",
		// 	storeURL: "https://foobar.docker.io",
		// 	readURL:  "https://foobar.docker.io:5678",
		// },
		{
			doc:      "non-matching paths",
			storeURL: "https://foobar.docker.io:1234/one/two",
			readURL:  "https://foobar.docker.io:1234/five/six",
		},
	}

	helper := Wincred{}
	t.Cleanup(func() {
		for _, tc := range tests {
			if err := helper.Delete(tc.storeURL); err != nil && !credentials.IsErrCredentialsNotFound(err) {
				t.Errorf("cleanup: failed to delete '%s': %v", tc.storeURL, err)
			}
		}
	})

	// Clean store before testing.
	for _, tc := range tests {
		if err := helper.Delete(tc.storeURL); err != nil && !credentials.IsErrCredentialsNotFound(err) {
			t.Errorf("prepare: failed to delete '%s': %v", tc.storeURL, err)
		}
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			c := &credentials.Credentials{ServerURL: tc.storeURL, Username: "hello", Secret: "world"}
			if err := helper.Add(c); err != nil {
				t.Fatalf("Error: failed to store secret for URL %q: %s", tc.storeURL, err)
			}
			if _, _, err := helper.Get(tc.readURL); err == nil {
				t.Errorf("Error: managed to read secret for URL %q using %q, but should not be able to", tc.storeURL, tc.readURL)
			}
			if err := helper.Delete(tc.storeURL); err != nil {
				t.Error(err)
			}
		})
	}
}

// TestWinCredHelperStoreRetrieve verifies that secrets stored in the
// the keychain can be read back using the URL that was used to store them.
func TestWinCredHelperStoreRetrieve(t *testing.T) {
	tests := []struct {
		url string
	}{
		{url: "foobar.docker.io"},
		{url: "foobar.docker.io:2376"},
		{url: "//foobar.docker.io:2376"},
		{url: "https://foobar.docker.io:2376"},
		{url: "http://foobar.docker.io:2376"},
		{url: "https://foobar.docker.io:2376/some/path"},
		{url: "https://foobar.docker.io:2376/some/other/path"},
		{url: "https://foobar.docker.io:2376/some/other/path?foo=bar"},
	}

	helper := Wincred{}
	t.Cleanup(func() {
		for _, tc := range tests {
			if err := helper.Delete(tc.url); err != nil && !credentials.IsErrCredentialsNotFound(err) {
				t.Errorf("cleanup: failed to delete '%s': %v", tc.url, err)
			}
		}
	})

	// Clean store before testing.
	for _, tc := range tests {
		if err := helper.Delete(tc.url); err != nil && !credentials.IsErrCredentialsNotFound(err) {
			t.Errorf("prepare: failed to delete '%s': %v", tc.url, err)
		}
	}

	// Note that we don't delete between individual tests here, to verify that
	// subsequent stores/overwrites don't affect storing / retrieving secrets.
	for i, tc := range tests {
		tc := tc
		t.Run(tc.url, func(t *testing.T) {
			c := &credentials.Credentials{
				ServerURL: tc.url,
				Username:  fmt.Sprintf("user-%d", i),
				Secret:    fmt.Sprintf("secret-%d", i),
			}

			if err := helper.Add(c); err != nil {
				t.Fatalf("Error: failed to store secret for URL: %s: %s", tc.url, err)
			}
			user, secret, err := helper.Get(tc.url)
			if err != nil {
				t.Fatalf("Error: failed to read secret for URL %q: %s", tc.url, err)
			}
			if user != c.Username {
				t.Errorf("Error: expected username %s, got username %s for URL: %s", c.Username, user, tc.url)
			}
			if secret != c.Secret {
				t.Errorf("Error: expected secret %s, got secret %s for URL: %s", c.Secret, secret, tc.url)
			}
		})
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Wincred{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
