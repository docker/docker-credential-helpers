package wincred

import (
	winc "github.com/danieljoos/wincred"
	"github.com/docker/docker-credential-helpers/credentials"
)

type wincred struct{}

// New creates a new wincred.
func New() credentials.Helper {
	return wincred{}
}

// Add adds new credentials to the windows credentials manager.
func (h wincred) Add(creds *credentials.Credentials) error {
	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.CredentialBlob = []byte(creds.Password)
	g.Persist = winc.PersistLocalMachine
	return g.Write()
}

// Delete removes credentials from the windows credentials manager.
func (h wincred) Delete(serverURL string) error {
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
func (h wincred) Get(serverURL string) (string, string, error) {
	g, _ := winc.GetGenericCredential(serverURL)
	if g == nil {
		return "", "", credentials.ErrCredentialsNotFound
	}
	return g.UserName, string(g.CredentialBlob), nil
}
