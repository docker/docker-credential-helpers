package pass

import (
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestPassHelper(t *testing.T) {
	helper := Pass{}

	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
		Username:  "nothing",
		Secret:    "isthebestmeshuggahalbum",
	}

	_ = helper.CheckInitialized()

	if err := helper.Add(creds); err != nil {
		t.Error(err)
	}

	creds.ServerURL = "https://foobar.docker.io:9999/v2"
	if err := helper.Add(creds); err != nil {
		t.Error(err)
	}

	credsList, err := helper.List()
	if err != nil {
		t.Fatal(err)
	}

	for server, username := range credsList {
		if !(strings.Contains(server, "2376") ||
			strings.Contains(server, "9999")) {
			t.Errorf("invalid url: %s", creds.ServerURL)
		}

		if username != "nothing" {
			t.Errorf("invalid username: %v", username)
		}

		u, s, err := helper.Get(server)
		if err != nil {
			t.Fatal(err)
		}

		if u != username {
			t.Errorf("invalid username %s", u)
		}

		if s != "isthebestmeshuggahalbum" {
			t.Errorf("invalid secret: %s", s)
		}

		err = helper.Delete(server)
		if err != nil {
			t.Error(err)
		}

		_, _, err = helper.Get(server)
		if !credentials.IsErrCredentialsNotFound(err) {
			t.Errorf("expected credentials not found, actual: %v", err)
		}
	}

	credsList, err = helper.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(credsList) != 0 {
		t.Error("didn't delete all creds?")
	}
}

func TestMissingCred(t *testing.T) {
	helper := Pass{}

	_, _, err := helper.Get("garbage")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected credentials not found, actual: %v", err)
	}
}
