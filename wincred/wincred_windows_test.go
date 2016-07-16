package wincred

import (
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

	paths, accts, err := helper.List()
	if err != nil || len(paths) == 0 || len(accts) == 0 {
		t.Fatal(err)
	}

	helper.Add(creds1)
	defer helper.Delete(creds1.ServerURL)
	newpaths, newaccts, err := helper.List()
	if len(newpaths)-len(paths) != 1 || len(newaccts)-len(accts) != 1 {
		if(err == nil) {
			t.Fatalf("Error: len(newpaths): %d, len(paths): %d\n len(newaccts): %d, len(accts): %d\n Error= %s", len(newpaths), len(paths), len(newaccts), len(accts), "")
		}
		t.Fatalf("Error: len(newpaths): %d, len(paths): %d\n len(newaccts): %d, len(accts): %d\n Error= %s", len(newpaths), len(paths), len(newaccts), len(accts), err.Error())
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Wincred{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
