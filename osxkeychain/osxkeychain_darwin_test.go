package osxkeychain

import (
	"testing"

	"github.com/calavera/docker-credential-helpers/credentials"
)

func TestOSXKeychainHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
		Username:  "foobar",
		Password:  "foobarbaz",
	}

	helper := New()
	if err := helper.Add(creds); err != nil {
		t.Fatal(err)
	}

	username, password, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Fatal(err)
	}

	if username != "foobar" {
		t.Fatalf("expected %s, got %s\n", "foobar", username)
	}

	if password != "foobarbaz" {
		t.Fatalf("expected %s, got %s\n", "foobarbaz", password)
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := New()
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if err != credentials.ErrCredentialsNotFound {
		t.Fatal("exptected ErrCredentialsNotFound, got %v", err)
	}
}
