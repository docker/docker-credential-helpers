package osxkeychain

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"testing"
)

func TestOSXKeychainHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
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

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}

	paths, accts, err := helper.List()
	if err != nil || len(paths) == 0 || len(accts) == 0 {
		t.Fatal(err)
	}
	helper.Add(creds)
	newpaths, newaccts, err := helper.List()
	if len(newpaths)-len(paths) != 1 || len(newaccts)-len(accts) != 1 {
		t.Fatal()
	}
	helper.Delete(creds.ServerURL)
}

func TestMissingCredentials(t *testing.T) {
	helper := Osxkeychain{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
