// Package gopass implements a `gopass` based credential helper. Passwords are
// stored as arguments to gopass of the form:
//
// "$GOPASS_FOLDER/base64-url(serverURL)/username"
//
// We base64-url encode the serverURL, because under the hood gopass uses files
// and folders, which would cause forward slasshes to get translated into
// additional folders.
package gopass

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/docker/docker-credential-helpers/credentials"
)

// GOPASS_FOLDER contains the directory where credentials are stored
const GOPASS_FOLDER = "docker-credential-helpers" //nolint:revive

// Gopass handles secrets using gopass as a store.
type Gopass struct{}

// Ideally these would be stored as members of Gopass, but since all of Gopass's
// methods have value receivers, not pointer receivers, and changing that is
// backwards incompatible, we assume that all Gopass instances share the same
// configuration

// initializationMutex is held while initializing so that only one 'gopass'
// round-tripping is done to check that gopass is functioning.
var initializationMutex sync.Mutex
var gopassInitialized bool

// CheckInitialized checks whether the password helper can be used. It
// internally caches and so may be safely called multiple times with no impact
// on performance, though the first call may take longer.
func (g Gopass) CheckInitialized() bool {
	return g.checkInitialized() == nil
}

func (g Gopass) checkInitialized() error {
	initializationMutex.Lock()
	defer initializationMutex.Unlock()
	if gopassInitialized {
		return nil
	}

	// We just run a `gopass ls`, if it fails then gopass is not initialized.
	_, err := g.runGopassHelper("", "ls", "--flat")
	if err != nil {
		return fmt.Errorf("gopass is not initialized: %v", err)
	}
	gopassInitialized = true
	return nil
}

func (g Gopass) runGopass(stdinContent string, args ...string) (string, error) {
	if err := g.checkInitialized(); err != nil {
		return "", err
	}
	return g.runGopassHelper(stdinContent, args...)
}

func (g Gopass) runGopassHelper(stdinContent string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gopass", args...)
	cmd.Stdin = strings.NewReader(stdinContent)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	// trim newlines; gopass includes a newline at the end of `show` output
	return strings.TrimRight(stdout.String(), "\n\r"), nil
}

// Add adds new credentials to the keychain.
func (g Gopass) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(creds.ServerURL))

	_, err := g.runGopass(creds.Secret, "insert", "-f", path.Join(GOPASS_FOLDER, encoded, creds.Username))
	return err
}

// Delete removes credentials from the store.
func (g Gopass) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))
	_, err := g.runGopass("", "rm", "-rf", path.Join(GOPASS_FOLDER, encoded))
	return err
}

func (g Gopass) getGopassDir() (string, error) {
	gopassDir, err := g.runGopass("", "config", "mounts.path")

	if err != nil {
		return "", fmt.Errorf("error getting gopass dir: %v", err)
	}

	ret := os.ExpandEnv(gopassDir)

	if strings.HasPrefix(ret, "~/") {
		d, err := os.UserHomeDir()

		if err != nil {
			message := fmt.Sprintf("unable to get user home directory: %v", err.Error())
			return "", errors.New(message)
		}

		ret = path.Join(d, ret[2:])
	}

	return ret, nil
}

// listGopassDir lists all the contents of a directory in the password store.
// Gopass uses fancy unicode to emit stuff to stdout, so rather than try
// and parse this, let's just look at the directory structure instead.
func (g Gopass) listGopassDir(args ...string) ([]os.FileInfo, error) {
	gopassDir, err := g.getGopassDir()
	if err != nil {
		return nil, err
	}

	p := os.ExpandEnv(path.Join(append([]string{gopassDir, GOPASS_FOLDER}, args...)...))

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
func (g Gopass) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	gopassDir, err := g.getGopassDir()
	if err != nil {
		return "", "", err
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))

	if _, err := os.Stat(path.Join(gopassDir, GOPASS_FOLDER, encoded)); err != nil {
		if os.IsNotExist(err) {
			return "", "", credentials.NewErrCredentialsNotFound()
		}

		return "", "", err
	}

	usernames, err := g.listGopassDir(encoded)
	if err != nil {
		return "", "", err
	}

	if len(usernames) < 1 {
		return "", "", fmt.Errorf("no usernames for %s", serverURL)
	}

	actual := strings.TrimSuffix(usernames[0].Name(), ".gpg")
	secret, err := g.runGopass("", "show", "-o", path.Join(GOPASS_FOLDER, encoded, actual))

	return actual, secret, err
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (g Gopass) List() (map[string]string, error) {
	servers, err := g.listGopassDir()
	if err != nil {
		return nil, err
	}

	resp := map[string]string{}

	for _, server := range servers {
		if !server.IsDir() {
			continue
		}

		serverURL, err := base64.URLEncoding.DecodeString(server.Name())
		if err != nil {
			return nil, err
		}

		usernames, err := g.listGopassDir(server.Name())
		if err != nil {
			return nil, err
		}

		if len(usernames) < 1 {
			return nil, fmt.Errorf("no usernames for %s", serverURL)
		}

		resp[string(serverURL)] = strings.TrimSuffix(usernames[0].Name(), ".gpg")
	}

	return resp, nil
}
