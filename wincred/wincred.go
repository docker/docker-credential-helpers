//go:build windows

package wincred

import (
	"bytes"
	"net/url"

	winc "github.com/danieljoos/wincred"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"
)

// Wincred handles secrets using the Windows credential service.
type Wincred struct{}

// Add adds new credentials to the windows credentials manager.
func (h Wincred) Add(creds *credentials.Credentials) error {
	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.CredentialBlob = []byte(creds.Secret)
	g.Persist = winc.PersistLocalMachine
	g.Attributes = append(g.Attributes, winc.CredentialAttribute{
		Keyword: "label",
		Value:   []byte(credentials.CredsLabel),
	})

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
			return g.UserName, string(g.CredentialBlob), nil
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
