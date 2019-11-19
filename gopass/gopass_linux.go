// A `gopass` based credential helper. Passwords are stored as arguments to gopass
// of the form: "$GOPASS_FOLDER/base64-url(serverURL)/username". We base64-url
// encode the serverURL, because under the hood gopass uses files and folders, so
// /s will get translated into additional folders.
package gopass

import (
	"bytes"
	// "encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/docker/docker-credential-helpers/credentials"
)

const GOPASS_FOLDER = "docker-credential-helpers"

// Gopass handles secrets using Linux secret-service as a store.
type Gopass struct{}

// Ideally these would be stored as members of Pass, but since all of Pass's
// methods have value receivers, not pointer receivers, and changing that is
// backwards incompatible, we assume that all Pass instances share the same configuration

// initializationMutex is held while initializing so that only one 'gopass'
// round-tripping is done to check gopass is functioning.
var initializationMutex sync.Mutex
var gopassInitialized bool

// CheckInitialized checks whether the password helper can be used. It
// internally caches and so may be safely called multiple times with no impact
// on performance, though the first call may take longer.
func (p Gopass) CheckInitialized() bool {
	return p.checkInitialized() == nil
}

func (p Gopass) checkInitialized() error {
	initializationMutex.Lock()
	defer initializationMutex.Unlock()
	if gopassInitialized {
		return nil
	}
	// We just run a `pass ls`, if it fails then pass is not initialized.
	_, err := p.runPassHelper("", "ls")
	if err != nil {
		return fmt.Errorf("gopass not initialized: %v", err)
	}
	gopassInitialized = true
	return nil
}

func (p Gopass) runPass(stdinContent string, args ...string) (string, error) {
	if err := p.checkInitialized(); err != nil {
		return "", err
	}
	return p.runPassHelper(stdinContent, args...)
}

func (p Gopass) runPassHelper(stdinContent string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gopass", args...)
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
func (h Gopass) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	// encoded := base64.URLEncoding.EncodeToString([]byte(creds.ServerURL))

	_, err := h.runPass(creds.Secret, "insert", "-f", "-m", path.Join(GOPASS_FOLDER, creds.ServerURL, creds.Username))
	return err
}

// Delete removes credentials from the store.
func (h Gopass) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}

	// encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))
	_, err := h.runPass("", "rm", "-r", "-f", path.Join(GOPASS_FOLDER, serverURL))
	return err
}

func getPassDir() string {
	passDir := "$HOME/.password-store"
	if envDir := os.Getenv("PASSWORD_STORE_DIR"); envDir != "" {
		passDir = envDir
	}
	return os.ExpandEnv(passDir)
}

// listPassDir lists all the contents of a directory in the password store.
// Gopass uses fancy unicode to emit stuff to stdout, so rather than try
// and parse this, let's just look at the directory structure instead.
func listPassDir(args ...string) ([]os.FileInfo, error) {
	passDir := getPassDir()
	p := path.Join(append([]string{passDir, GOPASS_FOLDER}, args...)...)
	contents, err := ioutil.ReadDir(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []os.FileInfo{}, nil
		}

		return nil, err
	}

	return contents, nil
}

// Get returns the username and secret to use for a given registry server URL.
func (h Gopass) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	// encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))

	if _, err := os.Stat(path.Join(getPassDir(), GOPASS_FOLDER, serverURL)); err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}

		return "", "", err
	}

	usernames, err := listPassDir(serverURL)
	if err != nil {
		return "", "", err
	}

	if len(usernames) < 1 {
		return "", "", fmt.Errorf("no usernames for %s", serverURL)
	}

	actual := strings.TrimSuffix(usernames[0].Name(), ".gpg")
	secret, err := h.runPass("", "show", path.Join(GOPASS_FOLDER, serverURL, actual))
	return actual, secret, err
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (h Gopass) List() (map[string]string, error) {
	servers, err := listPassDir()
	if err != nil {
		return nil, err
	}

	resp := map[string]string{}

	for _, server := range servers {
		if !server.IsDir() {
			continue
		}

		//serverURL, err := base64.URLEncoding.DecodeString(server.Name())
		if err != nil {
			return nil, err
		}

		usernames, err := listPassDir(server.Name())
		if err != nil {
			return nil, err
		}

		if len(usernames) < 1 {
			return nil, fmt.Errorf("no usernames for %s", server.Name())
		}

		resp[string(server.Name())] = strings.TrimSuffix(usernames[0].Name(), ".gpg")
	}

	return resp, nil
}
