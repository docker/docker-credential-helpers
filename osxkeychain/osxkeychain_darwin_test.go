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
	creds1 := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v2",
		Username:  "foobarbaz",
		Secret:    "foobar",
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

	helper.Add(creds1)
	defer helper.Delete(creds1.ServerURL)
	newauths, err := helper.List()
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

func TestOSXKeychainHelperIgnoresProtocol(t *testing.T) {
	loginServerURL := "foobar.docker.io:2376"
	pullServerURL1 := "http://foobar.docker.io:2376"
	pullServerURL2 := "https://foobar.docker.io:2376"
	pullServerURL3 := "ftp://foobar.docker.io:2376"

	creds := &credentials.Credentials{
		ServerURL: loginServerURL,
		Username:  "foobar",
		Secret:    "foobarbaz",
	}
	helper := Osxkeychain{}
	defer helper.Delete(creds.ServerURL)
	if err := helper.Add(creds); err != nil {
		t.Fatal(err)
	}

	username1, secret1, err := helper.Get(pullServerURL1)
	if err != nil {
		t.Fatal(err)
	}
	if username1 != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", creds.Username, username1)
	}
	if secret1 != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", creds.Secret, secret1)
	}

	username2, secret2, err := helper.Get(pullServerURL2)
	if err != nil {
		t.Fatal(err)
	}
	if username2 != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", creds.Username, username2)
	}
	if secret2 != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", creds.Secret, secret2)
	}

	username3, secret3, err := helper.Get(pullServerURL3)
	if err != nil {
		t.Fatal(err)
	}
	if username3 != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", creds.Username, username3)
	}
	if secret3 != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", creds.Secret, secret3)
	}

	username, secret, err := helper.Get(loginServerURL)
	if err != nil {
		t.Fatal(err)
	}
	if username != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", creds.Username, username)
	}
	if secret != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", creds.Secret, secret)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Osxkeychain{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
