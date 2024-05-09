//go:build !windows

package pass

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestPassHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.example.com:2376/v1",
		Username:  "nothing",
		Secret:    "isthebestmeshuggahalbum",
	}

	helper := Pass{}
	if err := helper.checkInitialized(); err != nil {
		t.Error(err)
	}

	if err := helper.Add(creds); err != nil {
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

func TestPassHelperCheckInit(t *testing.T) {
	helper := Pass{}
	if v := helper.CheckInitialized(); !v {
		t.Errorf("expected true, actual: %v", v)
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

// TestPassHelperWithEmptyServer verifies that empty directories (servers
// without credentials) are ignored, but still returns credentials for other
// servers.
func TestPassHelperWithEmptyServer(t *testing.T) {
	helper := Pass{}
	if err := helper.checkInitialized(); err != nil {
		t.Error(err)
	}

	creds := []*credentials.Credentials{
		{
			ServerURL: "https://myreqistry.example.com:2375/v1",
			Username:  "foo",
			Secret:    "isthebestmeshuggahalbum",
		},
		{
			ServerURL: "https://index.example.com/v1//access-token",
		},
	}

	t.Cleanup(func() {
		for _, cred := range creds {
			_ = helper.Delete(cred.ServerURL)
		}
	})

	for _, cred := range creds {
		if cred.Username != "" {
			if err := helper.Add(cred); err != nil {
				t.Error(err)
			}
		} else {
			// No credentials; create an empty directory for this server.
			serverURL := encodeServerURL(cred.ServerURL)
			p := path.Join(getPassDir(), PASS_FOLDER, serverURL)
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Error(err)
			}
		}
	}

	credsList, err := helper.List()
	if err != nil {
		t.Error(err)
	}
	if len(credsList) == 0 {
		t.Error("expected credentials to be returned, but got none")
	}
	for _, cred := range creds {
		if cred.Username != "" {
			userName, secret, err := helper.Get(cred.ServerURL)
			if err != nil {
				t.Error(err)
			}
			if userName != cred.Username {
				t.Errorf("expected username %q, actual: %q", cred.Username, userName)
			}
			if secret != cred.Secret {
				t.Errorf("expected secret %q, actual: %q", cred.Secret, secret)
			}
		} else {
			_, _, err := helper.Get(cred.ServerURL)
			if !credentials.IsErrCredentialsNotFound(err) {
				t.Errorf("expected credentials not found, actual: %v", err)
			}
		}
	}
}

func TestMissingCred(t *testing.T) {
	helper := Pass{}
	if _, _, err := helper.Get("garbage"); !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected credentials not found, actual: %v", err)
	}
}
