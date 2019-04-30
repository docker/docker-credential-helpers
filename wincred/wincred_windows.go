package wincred

import (
	"bytes"
	"net/url"
	"strings"

	winc "github.com/danieljoos/wincred"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"
)

// Wincred handles secrets using the Windows credential service.
type Wincred struct{}

// Add adds new credentials to the windows credentials manager.
func (h Wincred) Add(creds *credentials.Credentials) error {
	credsLabels := []byte(credentials.CredsLabel)
	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.CredentialBlob = []byte(creds.Secret)
	g.Persist = winc.PersistLocalMachine
	g.Attributes = []winc.CredentialAttribute{{Keyword: "label", Value: credsLabels}}

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
		if strings.Compare(attr.Keyword, "label") == 0 &&
			bytes.Compare(attr.Value, []byte(credentials.CredsLabel)) == 0 {

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
	for i := range creds {
		attrs := creds[i].Attributes
		for _, attr := range attrs {
			if attr.Keyword == "label" && bytes.Equal(attr.Value, []byte(credentials.CredsLabel)) {
				targets = append(targets, creds[i].TargetName)
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
	//if scheme is missing assume it is the same as target
	if serverURL.Scheme == "" {
		serverURL.Scheme = target.Scheme
	}
	//if port is missing assume it is the same as target
	if serverURL.Port() == "" && target.Port() != "" {
		serverURL.Host = serverURL.Host + ":" + target.Port()
	}
	//if path is missing assume it is the same as target
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
	for i := range creds {
		attrs := creds[i].Attributes
		for _, attr := range attrs {
			if strings.Compare(attr.Keyword, "label") == 0 &&
				bytes.Compare(attr.Value, []byte(credentials.CredsLabel)) == 0 {

				resp[creds[i].TargetName] = creds[i].UserName
			}
		}

	}

	return resp, nil
}
