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

// osxKeychainHelperGet wraps an operation of creating, adding, getting and deleting credentials for the provided
// username, secret and server URL
func osxKeychainHelperGet(username, secret, serverURL string) (string, string, error) {
	creds := &credentials.Credentials{
		ServerURL: serverURL,
		Username:  username,
		Secret:    secret,
	}

	helper := Osxkeychain{}
	defer helper.Delete(creds.ServerURL)
	if err := helper.Add(creds); err != nil {
		return "", "", err
	}

	return helper.Get(creds.ServerURL)
}

func TestOSXKeychainHelperURLProtocols(t *testing.T) {
	pullServerNoProto := "foobar.docker.io:2376" // Is equivalent to "https://foobar.docker.io:2376"
	pullServerHTTP := "http://foobar.docker.io:2376"
	pullServerHTTPS := "https://foobar.docker.io:2376"
	pullServerFTP := "ftp://foobar.docker.io:2376" // Must fail as FTP is not supported

	user := "foobar"
	secret := "foobarbaz"

	usernameNoProto, secretNoProto, err := osxKeychainHelperGet(user, secret, pullServerNoProto)
	if err != nil {
		t.Fatal(err)
	}
	if usernameNoProto != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", user, usernameNoProto)
	}
	if secretNoProto != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", secret, secretNoProto)
	}

	usernameHTTP, secretHTTP, err := osxKeychainHelperGet(user, secret, pullServerHTTP)
	if err != nil {
		t.Fatal(err)
	}
	if usernameHTTP != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", user, usernameHTTP)
	}
	if secretHTTP != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", secret, secretHTTP)
	}

	usernameHTTPS, secretHTTPS, err := osxKeychainHelperGet(user, secret, pullServerHTTPS)
	if err != nil {
		t.Fatal(err)
	}
	if usernameHTTPS != "foobar" {
		t.Fatalf("Error: expected username %s, got username %s", user, usernameHTTPS)
	}
	if secretHTTP != "foobarbaz" {
		t.Fatalf("Error: expected secret %s, got secret %s", secret, secretHTTPS)
	}

	_, _, err = osxKeychainHelperGet(user, secret, pullServerFTP)
	if err == nil {
		t.Fatal("Error expected due to unsupported protocol for URL: %s", pullServerFTP)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Osxkeychain{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
