package client

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/skip"
)

func newShellCommands(t *testing.T) (string, client.ProgramFunc, client.ProgramFunc) {
	cmd := os.Getenv("TEST_BC_COMMAND")
	previousCmd := os.Getenv("TEST_BC_PREVIOUS_COMMAND")
	skip.If(t, cmd == "", "TEST_BC_COMMAND not set, skipping backward compatibility test")
	skip.If(t, previousCmd == "", "TEST_BC_PREVIOUS_COMMAND not set, skipping backward compatibility test")

	oldP := client.NewShellProgramFunc(previousCmd)
	newP := client.NewShellProgramFunc(cmd)

	oldName, oldErr := version(oldP)
	newName, newErr := version(newP)

	assert.NilError(t, oldErr)
	assert.NilError(t, newErr)
	assert.Equal(t, oldName, newName)

	return oldName, oldP, newP
}

func version(program client.ProgramFunc) (string, error) {
	cmd := program(credentials.ActionVersion)
	out, err := cmd.Output()
	t := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("error getting version - err: %v, out: `%s`", err, t)
	}
	parts := strings.Split(t, " ")
	return parts[0], nil
}

// TestGetFromNewVersion tests that a new version of the helper can read
// credentials stored by an older version of the helper.
func TestGetFromNewVersion(t *testing.T) {
	helperName, oldP, newP := newShellCommands(t)

	skip.If(t, helperName == "docker-credential-secretservice", "test requires gnome-keyring but CI doesn't have it")

	testcases := []struct {
		name      string
		serverURL string
	}{
		{
			name:      "with no path",
			serverURL: "https://registry.example.com/",
		},
		{
			name:      "with path",
			serverURL: "https://registry.example.com/v1/",
		},
		{
			name:      "with port",
			serverURL: "https://registry.example.com:5000/",
		},
		{
			name:      "with path and port",
			serverURL: "https://registry.example.com:5000/v1/",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.Store(oldP, &credentials.Credentials{
				ServerURL: tc.serverURL,
				Username:  "testuser",
				Secret:    "testsecret",
			})
			defer client.Erase(newP, tc.serverURL)
			assert.NilError(t, err)

			creds, err := client.Get(newP, tc.serverURL)
			assert.NilError(t, err)

			assert.Check(t, is.Equal(creds.Username, "testuser"))
			assert.Check(t, is.Equal(creds.Secret, "testsecret"))
		})
	}
}

// TestListFromNewVersion tests that a new version of the helper can list
// credentials stored by an older version of the helper.
func TestListFromNewVersion(t *testing.T) {
	helperName, oldP, newP := newShellCommands(t)

	skip.If(t, helperName == "docker-credential-secretservice", "test requires gnome-keyring but CI doesn't have it")

	priorCreds, err := client.List(newP)
	assert.NilError(t, err)
	// Capture the number of credentials before we store any new ones. When
	// tests are run on a developer's machine, they may have credentials
	// already stored in the helper, and we don't want to count those
	// against the helper's ability to list credentials.
	priorList := len(priorCreds)

	testCreds := []struct {
		credentials.Credentials
		skip       bool
		skipReason string
	}{
		{
			Credentials: credentials.Credentials{
				ServerURL: "https://registry.example.com/",
				Username:  "testuser1",
				Secret:    "testsecret1",
			},
		},
		{
			Credentials: credentials.Credentials{
				ServerURL: "https://registry.example.com/v1/",
				Username:  "testuser1",
				Secret:    "testsecret1",
			},
		},
		{
			Credentials: credentials.Credentials{
				ServerURL: "https://registry.example.com/v2/",
				Username:  "testuser2",
				Secret:    "testsecret2",
			},
		},
		{
			Credentials: credentials.Credentials{
				ServerURL: "https://registry.example.com:5000",
				Username:  "testuser1",
				Secret:    "testsecret1",
			},
		},
		{
			Credentials: credentials.Credentials{
				ServerURL: "https://registry.example.com:5000/v1/",
				Username:  "testuser1",
				Secret:    "testsecret1",
			},
			skip:       helperName == "docker-credential-osxkeychain",
			skipReason: "docker-credential-osxkeychain returns malformed URI when a port is specified",
		},
	}
	for i := range testCreds {
		creds := testCreds[i]

		err = client.Store(oldP, &creds.Credentials)
		defer client.Erase(oldP, creds.ServerURL)
		assert.NilError(t, err)
	}

	oldCreds, err := client.List(oldP)
	assert.NilError(t, err)
	t.Logf("credentials found by old version: %+v", oldCreds)
	assert.Check(t, is.Equal(len(oldCreds), priorList+len(testCreds)))

	newCreds, err := client.List(newP)
	assert.NilError(t, err)
	t.Logf("credentials found by new version: %+v", newCreds)
	assert.Check(t, is.Equal(len(newCreds), priorList+len(testCreds)))

	for _, tc := range testCreds {
		t.Run(tc.ServerURL, func(t *testing.T) {
			skip.If(t, tc.skip, tc.skipReason)
			if _, ok := oldCreds[tc.ServerURL]; !ok {
				t.Errorf("ServerURL=%q: no credentials stored in oldCreds, want one", tc.ServerURL)
				return
			}
			if _, ok := newCreds[tc.ServerURL]; !ok {
				t.Errorf("ServerURL=%q: no credentials stored in newCreds, want one", tc.ServerURL)
				return
			}
			assert.Check(t, is.Equal(newCreds[tc.ServerURL], tc.Username), "ServerURL=%q: got username=%q, want username %q", tc.ServerURL, newCreds[tc.ServerURL], tc.Username)
			assert.Check(t, is.Equal(newCreds[tc.ServerURL], oldCreds[tc.ServerURL]), "ServerURL=%q: got username=%q, want username %q", tc.ServerURL, newCreds[tc.ServerURL], tc.Username)
		})
	}
}

// TestReplaceFromNewVersion tests that a new version of the helper can
// credentials stored by an older version of the helper, and that both the old
// and new helpers can read the credentials.
func TestReplaceFromNewVersion(t *testing.T) {
	helperName, oldP, newP := newShellCommands(t)

	skip.If(t, helperName == "docker-credential-secretservice", "test requires gnome-keyring but CI doesn't have it")

	const serverURL = "https://registry.example.com/"
	err := client.Store(oldP, &credentials.Credentials{
		ServerURL: serverURL,
		Username:  "testuser1",
		Secret:    "testsecret1",
	})
	// defer client.Erase(oldP, serverURL)
	assert.NilError(t, err)

	err = client.Store(newP, &credentials.Credentials{
		ServerURL: serverURL,
		Username:  "testuser2",
		Secret:    "testsecret2",
	})
	// defer client.Erase(newP, serverURL)
	assert.NilError(t, err)

	creds, err := client.Get(oldP, serverURL)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(creds.Username, "testuser2"))
	assert.Check(t, is.Equal(creds.Secret, "testsecret2"))

	creds, err = client.Get(newP, serverURL)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(creds.Username, "testuser2"))
	assert.Check(t, is.Equal(creds.Secret, "testsecret2"))
}

// TestEraseFromNewVersion tests that a new version of the helper can erase
// credentials stored by an older version of the helper.
func TestEraseFromNewVersion(t *testing.T) {
	helperName, oldP, newP := newShellCommands(t)

	skip.If(t, helperName == "docker-credential-secretservice", "test requires gnome-keyring but CI doesn't have it")

	const serverURL = "https://registry.example.com/"
	err := client.Store(oldP, &credentials.Credentials{
		ServerURL: serverURL,
		Username:  "testuser1",
		Secret:    "testsecret1",
	})
	defer client.Erase(oldP, serverURL)
	assert.NilError(t, err)

	err = client.Erase(newP, serverURL)
	assert.NilError(t, err)

	creds, err := client.Get(oldP, serverURL)
	assert.ErrorIs(t, err, credentials.NewErrCredentialsNotFound(), "expected err %q; got err = nil, creds = %+v", credentials.NewErrCredentialsNotFound(), creds)
	assert.Check(t, is.Nil(creds))

	creds, err = client.Get(newP, serverURL)
	assert.ErrorIs(t, err, credentials.NewErrCredentialsNotFound(), "expected err %q; got err = nil, creds = %+v", credentials.NewErrCredentialsNotFound(), creds)
	assert.Check(t, is.Nil(creds))
}
