//go:build darwin && cgo

package osxkeychain

import (
	"fmt"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestOSXKeychainHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.example.com:2376/v1",
		Username:  "foobar",
		Secret:    "foobarbaz",
	}
	helper := Osxkeychain{}
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
	if err != nil || len(auths) == 0 {
		t.Fatal(err)
	}

	if _, ok := auths[creds.ServerURL]; !ok {
		t.Fatalf("server %s not found in list, got: %+v", creds.ServerURL, auths)
	}

	// Insert another token and check if it is in the list
	creds1 := &credentials.Credentials{
		ServerURL: "https://foobar.example.com:2376/v2",
		Username:  "foobarbaz",
		Secret:    "foobar",
	}
	helper.Add(creds1)
	defer helper.Delete(creds1.ServerURL)

	auths, err = helper.List()
	if err != nil {
		t.Fatalf("operation List failed: %+v", err)
	}

	if _, ok := auths[creds.ServerURL]; !ok {
		t.Fatalf("server %s not found in list, got: %+v", creds.ServerURL, auths)
	}
	if _, ok := auths[creds1.ServerURL]; !ok {
		t.Fatalf("server %s not found in list, got: %+v", creds1.ServerURL, auths)
	}

	// Delete the 1st token inserted
	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}

	auths, err = helper.List()
	if err != nil {
		t.Fatalf("operation List failed: %+v", err)
	}

	// First token should have been deleted
	if _, ok := auths[creds.ServerURL]; ok {
		t.Fatalf("server %s was not deleted, got: %+v", creds.ServerURL, auths)
	}
	// Second token should still be there
	if _, ok := auths[creds1.ServerURL]; !ok {
		t.Fatalf("server %s not found in list, got: %+v", creds1.ServerURL, auths)
	}
}

// TestOSXKeychainHelperRetrieveAliases verifies that secrets can be accessed
// through variations on the URL
func TestOSXKeychainHelperRetrieveAliases(t *testing.T) {
	tests := []struct {
		doc      string
		storeURL string
		readURL  string
	}{
		{
			doc:      "stored with port, retrieved without",
			storeURL: "https://foobar.example.com:2376",
			readURL:  "https://foobar.example.com",
		},
		{
			doc:      "stored as https, retrieved without scheme",
			storeURL: "https://foobar.example.com:2376",
			readURL:  "foobar.example.com",
		},
		{
			doc:      "stored with path, retrieved without",
			storeURL: "https://foobar.example.com:1234/one/two",
			readURL:  "https://foobar.example.com:1234",
		},
	}

	helper := Osxkeychain{}
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
				t.Errorf("Error: failed to read secret for URL %q using %q: %s", tc.storeURL, tc.readURL, err)
			}
			if err := helper.Delete(tc.storeURL); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestOSXKeychainHelperStoreWithUncleanPath(t *testing.T) {
	helper := Osxkeychain{}
	creds := &credentials.Credentials{
		ServerURL: "https://::1:8080//////location/../../hello",
		Username:  "testuser",
		Secret:    "testsecret",
	}

	// Clean store before and after the test.
	defer helper.Delete(creds.ServerURL)
	if err := helper.Delete(creds.ServerURL); err != nil && !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("prepare: failed to delete '%s': %v", creds.ServerURL, err)
	}

	// Store the credentials
	if err := helper.Add(creds); err != nil {
		t.Fatalf("Error: failed to store credentials with unclean path %q: %s", creds.ServerURL, err)
	}

	// Retrieve and verify credentials
	username, secret, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Fatalf("Error: failed to retrieve credentials with unclean path %q: %s", creds.ServerURL, err)
	}

	if username != creds.Username {
		t.Errorf("Error: expected username %s, got %s", creds.Username, username)
	}
	if secret != creds.Secret {
		t.Errorf("Error: expected secret %s, got %s", creds.Secret, secret)
	}
}

// TestOSXKeychainHelperRetrieveStrict verifies that only matching secrets are
// returned.
func TestOSXKeychainHelperRetrieveStrict(t *testing.T) {
	tests := []struct {
		doc      string
		storeURL string
		readURL  string
	}{
		{
			doc:      "stored as https, retrieved using http",
			storeURL: "https://foobar.example.com:2376",
			readURL:  "http://foobar.example.com:2376",
		},
		{
			doc:      "stored as http, retrieved using https",
			storeURL: "http://foobar.example.com:2376",
			readURL:  "https://foobar.example.com:2376",
		},
		{
			// stored as http, retrieved without a scheme specified (hence, using the default https://)
			doc:      "stored as http, retrieved without scheme",
			storeURL: "http://foobar.example.com",
			readURL:  "foobar.example.com:5678",
		},
		{
			doc:      "non-matching ports",
			storeURL: "https://foobar.example.com:1234",
			readURL:  "https://foobar.example.com:5678",
		},
		// TODO: is this desired behavior? The other way round does work
		// {
		// 	doc:      "non-matching ports (stored without port)",
		// 	storeURL: "https://foobar.example.com",
		// 	readURL:  "https://foobar.example.com:5678",
		// },
		{
			doc:      "non-matching paths",
			storeURL: "https://foobar.example.com:1234/one/two",
			readURL:  "https://foobar.example.com:1234/five/six",
		},
	}

	helper := Osxkeychain{}
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

// TestOSXKeychainHelperStoreRetrieve verifies that secrets stored in the
// the keychain can be read back using the URL that was used to store them.
func TestOSXKeychainHelperStoreRetrieve(t *testing.T) {
	tests := []struct {
		url string
	}{
		{url: "foobar.example.com"},
		{url: "foobar.example.com:2376"},
		{url: "//foobar.example.com:2376"},
		{url: "https://foobar.example.com:2376"},
		{url: "http://foobar.example.com:2376"},
		{url: "https://foobar.example.com:2376/some/path"},
		{url: "https://foobar.example.com:2376/some/other/path"},
		{url: "https://foobar.example.com:2376/some/other/path?foo=bar"},
	}

	helper := Osxkeychain{}
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
	const nonExistingCred = "https://adsfasdf.invalid/asdfsdddd"
	helper := Osxkeychain{}
	_, _, err := helper.Get(nonExistingCred)
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
	err = helper.Delete(nonExistingCred)
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}
