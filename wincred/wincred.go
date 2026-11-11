//go:build windows

package wincred

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"

	winc "github.com/danieljoos/wincred"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"
)

// Wincred handles secrets using the Windows credential service.
type Wincred struct{}

// Add adds new credentials to the windows credentials manager.
func (h Wincred) Add(creds *credentials.Credentials) error {
	blob, chunks, err := splitSecret([]byte(creds.Secret))
	if err != nil {
		return err
	}

	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.CredentialBlob = blob
	g.Persist = winc.PersistLocalMachine
	g.Attributes = append(g.Attributes, winc.CredentialAttribute{
		Keyword: "label",
		Value:   []byte(credentials.CredsLabel),
	})
	for index, chunk := range chunks {
		g.Attributes = append(g.Attributes, winc.CredentialAttribute{
			Keyword: secretChunkAttributePrefix + strconv.Itoa(index),
			Value:   chunk,
		})
	}

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
		if attr.Keyword == "label" && bytes.Equal(attr.Value, []byte(credentials.CredsLabel)) {
			secret, err := joinSecret(g.CredentialBlob, secretChunks(g.Attributes))
			if err != nil {
				return "", "", err
			}
			return g.UserName, string(secret), nil
		}
	}
	return "", "", credentials.NewErrCredentialsNotFound()
}

// secretChunks returns the credential attributes that hold parts of a secret
// that did not fit in the credential blob, keyed by their keyword.
func secretChunks(attributes []winc.CredentialAttribute) map[string][]byte {
	chunks := make(map[string][]byte, len(attributes))
	for _, attr := range attributes {
		if strings.HasPrefix(attr.Keyword, secretChunkAttributePrefix) {
			chunks[attr.Keyword] = attr.Value
		}
	}
	return chunks
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
			if attr.Keyword == "label" && bytes.Equal(attr.Value, []byte(credentials.CredsLabel)) {
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
			if attr.Keyword == "label" && bytes.Equal(attr.Value, []byte(credentials.CredsLabel)) {
				resp[cred.TargetName] = cred.UserName
				break
			}
		}
	}

	return resp, nil
}
