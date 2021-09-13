// Package keyctl implements a `keyctl` based credential helper. Passwords are stored
// in linux kernel keyring.
package keyctl

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/jsipprell/keyctl"
	"github.com/pkg/errors"
)

// Keyctl based credential helper looks for a default keyring inside
// session keyring. It does all operations inside the default keyring

const defaultKeyringName string = "keyctlCredsStore"
const persistent int = 1

// Keyctl handles secrets using Linux Kernel keyring mechanism
type Keyctl struct{}

func (k Keyctl) createDefaultPersistentKeyring() (string, error) {
	/* Create default persistent keyring. If the keyring for the user
	 * already exists, then it returns the id of the existing keyring
	 */
	var errout, out bytes.Buffer
	uid := os.Getuid()
	cmd := exec.Command("keyctl", "get_persistent", "@u", strconv.Itoa(uid))
	cmd.Stderr = &errout
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "cannot run keyctl command to create persistent keyring %+v: %s", err, errout.String())
	}
	persistentKeyringID := out.String()
	if err != nil {
		return "", errors.Wrapf(err, "cannot create or read persistent keyring %+v", err)
	}
	return persistentKeyringID, nil
}

func (k Keyctl) getDefaultCredsStoreFromPersistent() (keyctl.NamedKeyring, error) {
	var out, errout bytes.Buffer
	persistentKeyringID, err := k.createDefaultPersistentKeyring()
	if err != nil {
		return nil, errors.Wrap(err, "default persistent keyring cannot be created")
	}

	defaultSessionKeyring, err := keyctl.SessionKeyring()
	if err != nil {
		return nil, errors.New("errors getting session keyring")
	}

	defaultKeyring, err := keyctl.OpenKeyring(defaultSessionKeyring, defaultKeyringName)
	/* if already does not exist we create */
	if err != nil || defaultKeyring == nil {
		cmd := exec.Command("keyctl", "newring", defaultKeyringName, strings.TrimSuffix(persistentKeyringID, "\n"))
		cmd.Stdout = &out
		cmd.Stderr = &errout
		err := cmd.Run()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot run keyctl command to created credstore keyring %s %s %s", cmd.String(), errout.String(), out.String())
		}
	}
	/* Search for it again and return the default keyring*/
	defaultKeyring, err = keyctl.OpenKeyring(defaultSessionKeyring, defaultKeyringName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup default session keyring")
	}

	return defaultKeyring, nil
}

// getDefaultCredsStore is a helper function to get the default credsStore keyring
func (k Keyctl) getDefaultCredsStore() (keyctl.NamedKeyring, error) {
	if persistent == 1 {
		return k.getDefaultCredsStoreFromPersistent()
	}
	defaultSessionKeyring, err := keyctl.SessionKeyring()
	if err != nil {
		return nil, errors.Wrap(err, "error getting session keyring")
	}

	defaultKeyring, err := keyctl.OpenKeyring(defaultSessionKeyring, defaultKeyringName)
	if err != nil || defaultKeyring == nil {
		if defaultKeyring == nil {
			defaultKeyring, err = keyctl.CreateKeyring(defaultSessionKeyring, defaultKeyringName)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create default credsStore keyring")
			}
		}
	}

	if defaultKeyring == nil {
		return nil, errors.Wrap(errors.New(""), " nil credstore")
	}

	return defaultKeyring, nil
}

// Add adds new credentials to the keychain.
func (k Keyctl) Add(creds *credentials.Credentials) error {
	defaultKeyring, err := k.getDefaultCredsStore()
	if err != nil || defaultKeyring == nil {
		return errors.Wrapf(err, "failed to create credsStore entry for %s", creds.ServerURL)
	}

	// create a child keyring under default for given url
	encoded := base64.URLEncoding.EncodeToString([]byte(strings.TrimSuffix(creds.ServerURL, "\n")))
	urlKeyring, err := keyctl.CreateKeyring(defaultKeyring, encoded)
	if err != nil {
		return errors.Wrapf(err, "failed to create keyring for %s", creds.ServerURL)
	}

	_, err = urlKeyring.Add(creds.Username, []byte(creds.Secret))
	if err != nil {
		return errors.Wrapf(err, "failed to add creds to keryring for %s with error: %+v", creds.ServerURL, err)
	}
	return err
}

// searchHelper function searches for an url inside the default keyring.
func (k Keyctl) searchHelper(serverURL string) (keyctl.NamedKeyring, string, error) {
	defaultKeyring, err := k.getDefaultCredsStore()
	if err != nil || defaultKeyring == nil {
		return nil, "", fmt.Errorf("searchHelper failed: cannot read defaultCredsStore")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(strings.TrimSuffix(serverURL, "\n")))
	urlKeyring, err := keyctl.OpenKeyring(defaultKeyring, encoded)
	if err != nil {
		return nil, "", fmt.Errorf("error in reading credsStore for url %s", serverURL)
	}
	if urlKeyring == nil {
		return nil, "", fmt.Errorf("credsStore entry for suplied url %s not found", serverURL)
	}

	refs, err := keyctl.ListKeyring(urlKeyring)
	if err != nil {
		return nil, "", fmt.Errorf("key for server url not found")
	}
	if len(refs) < 1 {
		return nil, "", fmt.Errorf("no keys in keyring %s", urlKeyring.Name())
	}

	obj := refs[0]
	id, err := obj.Get()
	if err != nil {
		return nil, "", fmt.Errorf("key for server url not found")
	}

	info, err := id.Info()
	if err != nil {
		return nil, "", fmt.Errorf("cannot read info for url key")
	}

	return urlKeyring, info.Name, err
}

// Get returns the username and secret to use for a given registry server URL.
func (k Keyctl) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	serverURL = strings.TrimSuffix(serverURL, "\n")
	urlKeyring, searchData, err := k.searchHelper(serverURL)
	if err != nil {
		return "", "", errors.Wrapf(err, "url not found by searchHelper:  %s err: %v", serverURL, err)
	}
	key, err := urlKeyring.Search(searchData)
	if err != nil {
		return "", "", errors.Wrapf(err, "url not found in %+v", urlKeyring)
	}
	secret, err := key.Get()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to read credentials for %s:%s", serverURL, searchData)
	}

	return searchData, string(secret), nil
}

// Delete removes credentials from the store.
func (k Keyctl) Delete(serverURL string) error {
	serverURL = strings.TrimSuffix(serverURL, "\n")
	urlKeyring, searchData, err := k.searchHelper(serverURL)
	if err != nil {
		return errors.Wrapf(err, "cannot find server url %s", serverURL)
	}

	key, err := urlKeyring.Search(searchData)
	if err != nil {
		return err
	}

	err = key.Unlink()
	if err != nil {
		return err
	}

	refs, err := keyctl.ListKeyring(urlKeyring)
	if err != nil {
		fmt.Printf("cannot list keyring %s", urlKeyring.Name())
	}
	if len(refs) == 0 {
		keyctl.UnlinkKeyring(urlKeyring)
	} else {
		return errors.Wrapf(err, "Canot remove keyring as its not empty %s", urlKeyring.Name())
	}

	return err
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (k Keyctl) List() (map[string]string, error) {
	defaultKeyring, err := k.getDefaultCredsStore()
	if err != nil || defaultKeyring == nil {
		return nil, errors.Wrap(err, "List() failed: cannot read default credStore")
	}

	resp := map[string]string{}

	refs, err := keyctl.ListKeyring(defaultKeyring)
	if err != nil {
		return nil, err
	}

	for _, r := range refs {
		id, _ := r.Get()
		info, _ := id.Info()
		url, _ := base64.URLEncoding.DecodeString(info.Name)

		key, _ := keyctl.OpenKeyring(defaultKeyring, info.Name)
		innerRefs, _ := keyctl.ListKeyring(key)

		if len(innerRefs) < 1 {
			continue
		}
		k, _ := innerRefs[0].Get()
		i, _ := k.Info()
		resp[string(url)] = i.Name
	}
	return resp, nil
}
