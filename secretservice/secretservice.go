//go:build linux && cgo

package secretservice

import (
	"errors"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/keybase/dbus"
	"github.com/keybase/go-keychain/secretservice"
)

const (
	schemaAttr     = "xdg:schema"
	labelAttr      = "label"
	serverAttr     = "server"
	usernameAttr   = "username"
	dockerCliAttr  = "docker_cli"
	dockerCliValue = "1"
)

// Secretservice handles secrets using Linux secret-service as a store.
type Secretservice struct{}

// Add adds new credentials to the keychain.
func (h Secretservice) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	service, session, err := getSession()
	if err != nil {
		return err
	}
	defer service.CloseSession(session)

	if err := unlock(service); err != nil {
		return err
	}

	secret, err := session.NewSecret([]byte(creds.Secret))
	if err != nil {
		return err
	}

	return handleTimeout(func() error {
		_, err = service.CreateItem(
			secretservice.DefaultCollection,
			secretservice.NewSecretProperties("Registry credentials for "+creds.ServerURL, map[string]string{
				schemaAttr:    "io.docker.Credentials",
				labelAttr:     credentials.CredsLabel,
				serverAttr:    creds.ServerURL,
				usernameAttr:  creds.Username,
				dockerCliAttr: dockerCliValue,
			}),
			secret,
			secretservice.ReplaceBehaviorReplace,
		)
		return err
	})
}

// Delete removes credentials from the store.
func (h Secretservice) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}

	service, session, err := getSession()
	if err != nil {
		return err
	}
	defer service.CloseSession(session)

	items, err := getItems(service, map[string]string{
		serverAttr:    serverURL,
		dockerCliAttr: dockerCliValue,
	})
	if err != nil {
		return err
	} else if len(items) == 0 {
		return credentials.NewErrCredentialsNotFound()
	}

	return handleTimeout(func() error {
		return service.DeleteItem(items[0])
	})
}

// Get returns the username and secret to use for a given registry server URL.
func (h Secretservice) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	service, session, err := getSession()
	if err != nil {
		return "", "", err
	}
	defer service.CloseSession(session)

	if err := unlock(service); err != nil {
		return "", "", err
	}

	items, err := getItems(service, map[string]string{
		serverAttr:    serverURL,
		dockerCliAttr: dockerCliValue,
	})
	if err != nil {
		return "", "", err
	} else if len(items) == 0 {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	attrs, err := service.GetAttributes(items[0])
	if err != nil {
		return "", "", err
	}

	var secret []byte
	err = handleTimeout(func() error {
		var err error
		secret, err = service.GetSecret(items[0], *session)
		return err
	})
	if err != nil {
		return "", "", err
	}

	return attrs[usernameAttr], string(secret), nil
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (h Secretservice) List() (map[string]string, error) {
	service, session, err := getSession()
	if err != nil {
		return nil, err
	}
	defer service.CloseSession(session)

	items, err := getItems(service, map[string]string{
		dockerCliAttr: dockerCliValue,
	})
	if err != nil {
		return nil, err
	}

	resp := make(map[string]string)
	if len(items) == 0 {
		return resp, nil
	}

	for _, it := range items {
		attrs, err := service.GetAttributes(it)
		if err != nil {
			return nil, err
		}
		if v, ok := attrs[usernameAttr]; !ok || v == "" {
			continue
		}
		resp[attrs[serverAttr]] = attrs[usernameAttr]
	}

	return resp, nil
}

func getSession() (*secretservice.SecretService, *secretservice.Session, error) {
	service, err := secretservice.NewService()
	if err != nil {
		return nil, nil, err
	}
	session, err := service.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return nil, nil, err
	}
	return service, session, nil
}

func unlock(service *secretservice.SecretService) error {
	return handleTimeout(func() error {
		return service.Unlock([]dbus.ObjectPath{secretservice.DefaultCollection})
	})
}

func handleTimeout(f func() error) error {
	err := f()
	if errors.Is(err, errors.New("prompt timed out")) {
		return f()
	}
	return err
}

func getItems(service *secretservice.SecretService, attributes map[string]string) ([]dbus.ObjectPath, error) {
	if err := unlock(service); err != nil {
		return nil, err
	}

	var items []dbus.ObjectPath
	err := handleTimeout(func() error {
		var err error
		items, err = service.SearchCollection(
			secretservice.DefaultCollection,
			attributes,
		)
		return err
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}
