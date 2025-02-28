//go:build darwin && cgo

package osxkeychain

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation

#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
*/
import "C"

import (
	"errors"
	"strconv"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"
	"github.com/keybase/go-keychain"
)

// https://opensource.apple.com/source/Security/Security-55471/sec/Security/SecBase.h.auto.html
const (
	// errCredentialsNotFound is the specific error message returned by OS X
	// when the credentials are not in the keychain.
	errCredentialsNotFound = "The specified item could not be found in the keychain. (-25300)"
	// errInteractionNotAllowed is the specific error message returned by OS X
	// when environment does not allow showing dialog to unlock keychain.
	errInteractionNotAllowed = "User interaction is not allowed. (-25308)"
)

// ErrInteractionNotAllowed is returned if keychain password prompt can not be shown.
var ErrInteractionNotAllowed = errors.New(`keychain cannot be accessed because the current session does not allow user interaction. The keychain may be locked; unlock it by running "security -v unlock-keychain ~/Library/Keychains/login.keychain-db" and try again`)

// Osxkeychain handles secrets using the OS X Keychain as store.
type Osxkeychain struct{}

// Add adds new credentials to the keychain.
func (h Osxkeychain) Add(creds *credentials.Credentials) error {
	_ = h.Delete(creds.ServerURL) // ignore errors as existing credential may not exist.

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassInternetPassword)
	item.SetLabel(credentials.CredsLabel)
	item.SetAccount(creds.Username)
	item.SetData([]byte(creds.Secret))
	if err := splitServer(creds.ServerURL, item); err != nil {
		return err
	}

	return keychain.AddItem(item)
}

// Delete removes credentials from the keychain.
func (h Osxkeychain) Delete(serverURL string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassInternetPassword)
	if err := splitServer(serverURL, item); err != nil {
		return err
	}
	if err := keychain.DeleteItem(item); err != nil {
		switch err.Error() {
		case errCredentialsNotFound:
			return credentials.NewErrCredentialsNotFound()
		case errInteractionNotAllowed:
			return ErrInteractionNotAllowed
		default:
			return err
		}
	}
	return nil
}

// Get returns the username and secret to use for a given registry server URL.
func (h Osxkeychain) Get(serverURL string) (string, string, error) {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassInternetPassword)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetReturnAttributes(true)
	item.SetReturnData(true)
	if err := splitServer(serverURL, item); err != nil {
		return "", "", err
	}

	res, err := keychain.QueryItem(item)
	if err != nil {
		switch err.Error() {
		case errCredentialsNotFound:
			return "", "", credentials.NewErrCredentialsNotFound()
		case errInteractionNotAllowed:
			return "", "", ErrInteractionNotAllowed
		default:
			return "", "", err
		}
	} else if len(res) == 0 {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	return res[0].Account, string(res[0].Data), nil
}

// List returns the stored URLs and corresponding usernames.
func (h Osxkeychain) List() (map[string]string, error) {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassInternetPassword)
	item.SetMatchLimit(keychain.MatchLimitAll)
	item.SetReturnAttributes(true)
	item.SetLabel(credentials.CredsLabel)

	res, err := keychain.QueryItem(item)
	if err != nil {
		switch err.Error() {
		case errCredentialsNotFound:
			return make(map[string]string), nil
		case errInteractionNotAllowed:
			return nil, ErrInteractionNotAllowed
		default:
			return nil, err
		}
	}

	resp := make(map[string]string)
	for _, r := range res {
		if r.Path == "" {
			continue
		}
		resp[r.Path] = r.Account
	}
	return resp, nil
}

const (
	// Hardcoded protocol types matching their Objective-C equivalents.
	// https://developer.apple.com/documentation/security/ksecattrprotocolhttps?language=objc
	kSecProtocolTypeHTTPS = "htps" // This is NOT a typo.
	// https://developer.apple.com/documentation/security/ksecattrprotocolhttp?language=objc
	kSecProtocolTypeHTTP = "http"
)

func splitServer(serverURL string, item keychain.Item) error {
	u, err := registryurl.Parse(serverURL)
	if err != nil {
		return err
	}
	item.SetProtocol(kSecProtocolTypeHTTPS)
	if u.Scheme == "http" {
		item.SetProtocol(kSecProtocolTypeHTTP)
	}
	item.SetServer(u.Hostname())
	if p := u.Port(); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return err
		}
		item.SetPort(int32(port))
	}
	item.SetPath(u.Path)
	return nil
}
