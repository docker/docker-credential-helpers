//go:build linux && cgo

package secretservice

import (
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestSecretServiceHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
		Username:  "foobar",
		Secret:    "foobarbaz",
	}

	helper := Secretservice{}

	// Check how many docker credentials we have when starting the test
	oldAuths, err := helper.List()
	if err != nil {
		t.Error(err)
	}

	// If any docker credentials with the tests values we are providing, we
	// remove them as they probably come from a previous failed test
	for k, v := range oldAuths {
		if strings.Compare(k, creds.ServerURL) == 0 && strings.Compare(v, creds.Username) == 0 {
			if err := helper.Delete(creds.ServerURL); err != nil {
				t.Error(err)
			}
		}
	}

	// Check again how many docker credentials we have when starting the test
	oldAuths, err = helper.List()
	if err != nil {
		t.Error(err)
	}

	// Add new credentials
	if err := helper.Add(creds); err != nil {
		t.Error(err)
	}

	// Verify that it is inside the secret service store
	username, secret, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Error(err)
	}

	if username != "foobar" {
		t.Errorf("expected %s, got %s\n", "foobar", username)
	}

	if secret != "foobarbaz" {
		t.Errorf("expected %s, got %s\n", "foobarbaz", secret)
	}

	// We should have one more credential than before adding
	newAuths, err := helper.List()
	if err != nil || (len(newAuths)-len(oldAuths) != 1) {
		t.Error(err)
	}
	oldAuths = newAuths

	// Deleting the credentials associated to current server url should succeed
	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Error(err)
	}

	// We should have one less credential than before deleting
	newAuths, err = helper.List()
	if err != nil || (len(oldAuths)-len(newAuths) != 1) {
		t.Error(err)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Secretservice{}
	if _, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd"); !credentials.IsErrCredentialsNotFound(err) {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}
