package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
)

// Store uses an external program to save credentials.
func Store(program ProgramFunc, creds *credentials.Credentials) error {
	cmd := program("store")

	buffer := new(bytes.Buffer)
	if err := json.NewEncoder(buffer).Encode(creds); err != nil {
		return err
	}
	cmd.Input(buffer)

	out, err := cmd.Output()
	if err != nil {
		t := strings.TrimSpace(string(out))

		if credentials.IsCredentialsMissingServerURLMessage(t) {
			return credentials.NewErrCredentialsMissingServerURL()
		}

		if credentials.IsCredentialsMissingUsernameMessage(t) {
			return credentials.NewErrCredentialsMissingUsername()
		}

		return fmt.Errorf("error storing credentials - err: %v, out: `%s`", err, t)
	}

	return nil
}

// Get executes an external program to get the credentials from a native store.
func Get(program ProgramFunc, serverURL string) (*credentials.Credentials, error) {
	cmd := program("get")
	cmd.Input(strings.NewReader(serverURL))

	out, err := cmd.Output()
	if err != nil {
		t := strings.TrimSpace(string(out))

		if credentials.IsCredentialsMissingServerURLMessage(t) {
			return nil, credentials.NewErrCredentialsMissingServerURL()
		}

		if credentials.IsCredentialsMissingUsernameMessage(t) {
			return nil, credentials.NewErrCredentialsMissingUsername()
		}

		if credentials.IsErrCredentialsNotFoundMessage(t) {
			return nil, credentials.NewErrCredentialsNotFound()
		}

		return nil, fmt.Errorf("error getting credentials - err: %v, out: `%s`", err, t)
	}

	resp := &credentials.Credentials{
		ServerURL: serverURL,
	}

	if err := json.NewDecoder(bytes.NewReader(out)).Decode(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// Erase executes a program to remove the server credentials from the native store.
func Erase(program ProgramFunc, serverURL string) error {
	cmd := program("erase")
	cmd.Input(strings.NewReader(serverURL))
	out, err := cmd.Output()
	if err != nil {
		t := strings.TrimSpace(string(out))

		if credentials.IsCredentialsMissingServerURLMessage(t) {
			return credentials.NewErrCredentialsMissingServerURL()
		}

		if credentials.IsCredentialsMissingUsernameMessage(t) {
			return credentials.NewErrCredentialsMissingUsername()
		}

		return fmt.Errorf("error erasing credentials - err: %v, out: `%s`", err, t)
	}

	return nil
}

// List executes a program to list server credentials in the native store.
func List(program ProgramFunc) (map[string]string, error) {
	cmd := program("list")
	cmd.Input(strings.NewReader("unused"))
	out, err := cmd.Output()
	if err != nil {
		t := strings.TrimSpace(string(out))

		if credentials.IsCredentialsMissingServerURLMessage(t) {
			return nil, credentials.NewErrCredentialsMissingServerURL()
		}

		if credentials.IsCredentialsMissingUsernameMessage(t) {
			return nil, credentials.NewErrCredentialsMissingUsername()
		}

		return nil, fmt.Errorf("error listing credentials - err: %v, out: `%s`", err, t)
	}

	var resp map[string]string
	if err = json.NewDecoder(bytes.NewReader(out)).Decode(&resp); err != nil {
		return nil, err
	}

	return resp, nil
}
