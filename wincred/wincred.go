//go:build windows

package wincred

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"

	winc "github.com/danieljoos/wincred"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"
)

var credsLabel = []byte(credentials.CredsLabel)

// Wincred handles secrets using the Windows credential service.
type Wincred struct{}

// Add adds new credentials to the windows credentials manager.
func (h Wincred) Add(creds *credentials.Credentials) error {
	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.Persist = winc.PersistLocalMachine
	g.Attributes = []winc.CredentialAttribute{
		{Keyword: "label", Value: credsLabel},
		{Keyword: "encoding", Value: []byte("utf16le")},
	}

	blob, err := encodeUTF16LE(creds.Secret)
	if err != nil {
		return err
	}

	g.CredentialBlob = blob

	return g.Write()
}

// Delete removes credentials from the windows credentials manager.
func (h Wincred) Delete(serverURL string) error {
	g, err := winc.GetGenericCredential(serverURL)
	if g == nil {
		return nil
	}
	if err != nil {
		return err
	}
	return g.Delete()
}

// Get retrieves credentials from the windows credentials manager.
func (h Wincred) Get(serverURL string) (string, string, error) {
	target, err := getTarget(serverURL)
	if err != nil {
		return "", "", err
	} else if target == "" {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	g, _ := winc.GetGenericCredential(target)
	if g == nil {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	for _, attr := range g.Attributes {
		if attr.Keyword == "label" && bytes.Equal(attr.Value, credsLabel) {
			switch enc := credentialEncoding(g.Attributes); enc {
			case "utf16le":
				// Encoding was stored; only accept UTF-16LE or error otherwise.
				creds, err := decodeUTF16LE(g.CredentialBlob)
				if err != nil {
					return "", "", fmt.Errorf("decoding credentials: %w", err)
				}
				return g.UserName, string(creds), nil
			case "":
				// Older versions of the wincred credential-helper stored the password blob
				// as raw string bytes. Newer versions store it as UTF-16LE. Try decoding from
				// UTF-16LE, otherwise assume creds were stored as raw bytes.
				//
				// This could also be the case if an external tool stored the credentials and
				// did not set the "encoding" attribute.
				//
				// See https://github.com/docker/docker-credential-helpers/pull/335
				creds := string(g.CredentialBlob)
				if c, ok := tryDecodeUTF16LE(g.CredentialBlob); ok {
					creds = c
				}
				return g.UserName, creds, nil
			default:
				return "", "", errors.New("unsupported credential encoding: " + enc)
			}
		}
	}
	return "", "", credentials.NewErrCredentialsNotFound()
}

func getTarget(serverURL string) (string, error) {
	s, err := registryurl.Parse(serverURL)
	if err != nil {
		return serverURL, nil
	}

	creds, err := winc.List()
	if err != nil {
		return "", err
	}

	var targets []string
	for _, cred := range creds {
		for _, attr := range cred.Attributes {
			if attr.Keyword == "label" && bytes.Equal(attr.Value, credsLabel) {
				targets = append(targets, cred.TargetName)
			}
		}
	}

	if target, found := findMatch(s, targets, exactMatch); found {
		return target, nil
	}

	if target, found := findMatch(s, targets, approximateMatch); found {
		return target, nil
	}

	return "", nil
}

func findMatch(serverUrl *url.URL, targets []string, matches func(url.URL, url.URL) bool) (string, bool) {
	for _, target := range targets {
		tURL, err := registryurl.Parse(target)
		if err != nil {
			continue
		}
		if matches(*serverUrl, *tURL) {
			return target, true
		}
	}
	return "", false
}

func exactMatch(serverURL, target url.URL) bool {
	return serverURL.String() == target.String()
}

func approximateMatch(serverURL, target url.URL) bool {
	// if scheme is missing assume it is the same as target
	if serverURL.Scheme == "" {
		serverURL.Scheme = target.Scheme
	}
	// if port is missing assume it is the same as target
	if serverURL.Port() == "" && target.Port() != "" {
		serverURL.Host = serverURL.Host + ":" + target.Port()
	}
	// if path is missing assume it is the same as target
	if serverURL.Path == "" {
		serverURL.Path = target.Path
	}
	return serverURL.String() == target.String()
}

// List returns the stored URLs and corresponding usernames for a given credentials label.
func (h Wincred) List() (map[string]string, error) {
	creds, err := winc.List()
	if err != nil {
		return nil, err
	}

	resp := make(map[string]string)

	for _, cred := range creds {
		for _, attr := range cred.Attributes {
			if attr.Keyword == "label" && bytes.Equal(attr.Value, credsLabel) {
				resp[cred.TargetName] = cred.UserName
				break
			}
		}
	}

	return resp, nil
}

func credentialEncoding(attrs []winc.CredentialAttribute) string {
	for _, attr := range attrs {
		if attr.Keyword == "encoding" {
			return string(attr.Value)
		}
	}
	return ""
}

func tryDecodeUTF16LE(blob []byte) (string, bool) {
	if len(blob)%2 != 0 {
		return "", false
	}

	decoded, err := decodeUTF16LE(blob)
	if err != nil {
		return "", false
	}

	s := string(decoded)
	encoded, err := encodeUTF16LE(s)
	if err != nil {
		return "", false
	}

	// round-trip the value to verify it was indeed valid UTF-16LE.
	if !bytes.Equal(encoded, blob) {
		return "", false
	}

	return s, true
}

func decodeUTF16LE(blob []byte) ([]byte, error) {
	decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	decoded, _, err := transform.Bytes(decoder, blob)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func encodeUTF16LE(s string) ([]byte, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	encoded, _, err := transform.Bytes(encoder, []byte(s))
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
