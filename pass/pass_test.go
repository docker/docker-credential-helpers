//go:build !windows

package pass

import (
	"encoding/base64"
	"path"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestPassHelperCheckInit(t *testing.T) {
	helper := Pass{}
	if v := helper.CheckInitialized(); !v {
		t.Errorf("expected true, actual: %v", v)
	}
}

func TestPassHelper(t *testing.T) {
	tests := []struct {
		name  string
		creds *credentials.Credentials
	}{
		{
			name: "create nothing",
			creds: &credentials.Credentials{
				ServerURL: "https://foobar.docker.io:2376/v1",
				Username:  "nothing",
				Secret:    "isthebestmeshuggahalbum",
			},
		},
		{
			name: "create foo/bar",
			creds: &credentials.Credentials{
				ServerURL: "https://foobar.docker.io:2376/v1",
				Username:  "foo/bar",
				Secret:    "foobarbaz",
			},
		},
	}

	helper := Pass{}
	if err := helper.checkInitialized(); err != nil {
		t.Error(err)
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := helper.Add(tc.creds); err != nil {
				t.Error(err)
			}
			u, s, err := helper.Get(tc.creds.ServerURL)
			if err != nil {
				t.Error(err)
			}
			if u != tc.creds.Username {
				t.Errorf("invalid username %s", u)
			}
			if s != tc.creds.Secret {
				t.Errorf("invalid secret: %s", s)
			}
			if err := helper.Delete(tc.creds.ServerURL); err != nil {
				t.Error(err)
			}
			if _, _, err := helper.Get(tc.creds.ServerURL); !credentials.IsErrCredentialsNotFound(err) {
				t.Errorf("expected credentials not found, actual: %v", err)
			}
		})
	}
}

func TestPassHelperBackwardCompat(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.example.com:2376/v1",
		Username:  "nothing",
		Secret:    "isthebestmeshuggahalbum",
	}

	helper := Pass{}
	if err := helper.checkInitialized(); err != nil {
		t.Error(err)
	}

	// add a credential with the old format
	encodedServerURL := base64.URLEncoding.EncodeToString([]byte(creds.ServerURL))
	if _, err := helper.runPass(creds.Secret, "insert", "-f", "-m", path.Join(PASS_FOLDER, encodedServerURL, creds.Username)); err != nil {
		t.Error(err)
	}

	u, s, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Error(err)
	}
	if u != creds.Username {
		t.Errorf("invalid username %s", u)
	}
	if s != creds.Secret {
		t.Errorf("invalid secret: %s", s)
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Error(err)
	}
	if _, _, err := helper.Get(creds.ServerURL); !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected credentials not found, actual: %v", err)
	}
}

func TestPassHelperList(t *testing.T) {
	creds := []*credentials.Credentials{
		{
			ServerURL: "https://foobar.example.com:2376/v1",
			Username:  "foo",
			Secret:    "isthebestmeshuggahalbum",
		},
		{
			ServerURL: "https://foobar.example.com:2375/v1",
			Username:  "bar",
			Secret:    "isthebestmeshuggahalbum",
		},
	}

	helper := Pass{}
	if err := helper.checkInitialized(); err != nil {
		t.Error(err)
	}

	for _, cred := range creds {
		if err := helper.Add(cred); err != nil {
			t.Error(err)
		}
	}

	credsList, err := helper.List()
	if err != nil {
		t.Error(err)
	}
	for server, username := range credsList {
		if !(strings.HasSuffix(server, "2376/v1") || strings.HasSuffix(server, "2375/v1")) {
			t.Errorf("invalid url: %s", server)
		}
		if !(username == "foo" || username == "bar") {
			t.Errorf("invalid username: %v", username)
		}

		u, s, err := helper.Get(server)
		if err != nil {
			t.Error(err)
		}
		if u != username {
			t.Errorf("invalid username %s", u)
		}
		if s != "isthebestmeshuggahalbum" {
			t.Errorf("invalid secret: %s", s)
		}

		if err := helper.Delete(server); err != nil {
			t.Error(err)
		}
		if _, _, err := helper.Get(server); !credentials.IsErrCredentialsNotFound(err) {
			t.Errorf("expected credentials not found, actual: %v", err)
		}
	}

	credsList, err = helper.List()
	if err != nil {
		t.Error(err)
	}
	if len(credsList) != 0 {
		t.Error("didn't delete all creds?")
	}
}

func TestMissingCred(t *testing.T) {
	helper := Pass{}
	if _, _, err := helper.Get("garbage"); !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected credentials not found, actual: %v", err)
	}
}
