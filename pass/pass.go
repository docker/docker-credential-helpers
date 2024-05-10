// Package pass implements a `pass` based credential helper. Passwords are stored
// as arguments to pass of the form: "$PASS_FOLDER/base64-url(serverURL)/username".
// We base64-url encode the serverURL, because under the hood pass uses files and
// folders, so /s will get translated into additional folders.
package pass

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker-credential-helpers/credentials"
)

// PASS_FOLDER contains the directory where credentials are stored
const PASS_FOLDER = "docker-credential-helpers" //nolint:revive

// Pass handles secrets using pass as a store.
type Pass struct{}

// Ideally these would be stored as members of Pass, but since all of Pass's
// methods have value receivers, not pointer receivers, and changing that is
// backwards incompatible, we assume that all Pass instances share the same configuration
var (
	// initializationMutex is held while initializing so that only one 'pass'
	// round-tripping is done to check pass is functioning.
	initializationMutex sync.Mutex
	passInitialized     bool
)

// CheckInitialized checks whether the password helper can be used. It
// internally caches and so may be safely called multiple times with no impact
// on performance, though the first call may take longer.
func (p Pass) CheckInitialized() bool {
	return p.checkInitialized() == nil
}

func (p Pass) checkInitialized() error {
	initializationMutex.Lock()
	defer initializationMutex.Unlock()
	if passInitialized {
		return nil
	}
	// We just run a `pass ls`, if it fails then pass is not initialized.
	_, err := p.runPassHelper("", "ls")
	if err != nil {
		return fmt.Errorf("pass not initialized: %v", err)
	}
	passInitialized = true
	return nil
}

func (p Pass) runPass(stdinContent string, args ...string) (string, error) {
	if err := p.checkInitialized(); err != nil {
		return "", err
	}
	return p.runPassHelper(stdinContent, args...)
}

func (p Pass) runPassHelper(stdinContent string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("pass", args...)
	cmd.Stdin = strings.NewReader(stdinContent)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	// trim newlines; pass v1.7.1+ includes a newline at the end of `show` output
	return strings.TrimRight(stdout.String(), "\n\r"), nil
}

// Add adds new credentials to the keychain.
func (p Pass) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	encoded := encodeServerURL(creds.ServerURL)
	_, err := p.runPass(creds.Secret, "insert", "-f", "-m", path.Join(PASS_FOLDER, encoded, creds.Username))
	return err
}

// Delete removes credentials from the store.
func (p Pass) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}

	encoded := encodeServerURL(serverURL)
	_, err := p.runPass("", "rm", "-rf", path.Join(PASS_FOLDER, encoded))
	return err
}

func getPassDir() string {
	if passDir := os.Getenv("PASSWORD_STORE_DIR"); passDir != "" {
		return passDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".password-store")
}

// listPassDir lists all the contents of a directory in the password store.
// Pass uses fancy unicode to emit stuff to stdout, so rather than try
// and parse this, let's just look at the directory structure instead.
func listPassDir(args ...string) ([]os.FileInfo, error) {
	passDir := getPassDir()
	p := path.Join(append([]string{passDir, PASS_FOLDER}, args...)...)
	entries, err := os.ReadDir(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []os.FileInfo{}, nil
		}
		return nil, err
	}
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Get returns the username and secret to use for a given registry server URL.
func (p Pass) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	encoded := encodeServerURL(serverURL)
	usernames, err := listPassDir(encoded)
	if err != nil {
		return "", "", err
	}

	if len(usernames) < 1 {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	actual := strings.TrimSuffix(usernames[0].Name(), ".gpg")
	secret, err := p.runPass("", "show", path.Join(PASS_FOLDER, encoded, actual))
	return actual, secret, err
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (p Pass) List() (map[string]string, error) {
	servers, err := listPassDir()
	if err != nil {
		return nil, err
	}

	resp := map[string]string{}

	for _, server := range servers {
		if !server.IsDir() {
			continue
		}

		serverURL, err := decodeServerURL(server.Name())
		if err != nil {
			return nil, err
		}

		usernames, err := listPassDir(server.Name())
		if err != nil {
			return nil, err
		}

		if len(usernames) < 1 {
			continue
		}

		resp[serverURL] = strings.TrimSuffix(usernames[0].Name(), ".gpg")
	}

	return resp, nil
}

// encodeServerURL returns the serverURL in base64-URL encoding to use
// as directory-name in pass storage.
func encodeServerURL(serverURL string) string {
	return base64.URLEncoding.EncodeToString([]byte(serverURL))
}

// decodeServerURL decodes base64-URL encoded serverURL. ServerURLs are
// used in encoded format for directory-names in pass storage.
func decodeServerURL(encodedServerURL string) (string, error) {
	serverURL, err := base64.URLEncoding.DecodeString(encodedServerURL)
	if err != nil {
		return "", err
	}
	return string(serverURL), nil
}
