package client

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

const (
	validServerAddress   = "https://registry.example.com/v1"
	validUsername        = "linus"
	validServerAddress2  = "https://example.com:5002"
	invalidServerAddress = "https://foobar.example.com"
	missingCredsAddress  = "https://missing.example.com/v1"
)

var errProgramExited = fmt.Errorf("exited 1")

// mockProgram simulates interactions between the docker client and a remote
// credentials-helper.
// Unit tests inject this mocked command into the remote to control execution.
type mockProgram struct {
	arg   string
	input io.Reader
}

// Output returns responses from the remote credentials-helper.
// It mocks those responses based in the input in the mock.
func (m *mockProgram) Output() ([]byte, error) {
	in, err := io.ReadAll(m.input)
	if err != nil {
		return nil, err
	}
	inS := string(in)

	switch m.arg {
	case "erase":
		switch inS {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errProgramExited
		}
	case "get":
		switch inS {
		case validServerAddress:
			return []byte(`{"Username": "foo", "Secret": "bar"}`), nil
		case validServerAddress2:
			return []byte(`{"Username": "<token>", "Secret": "abcd1234"}`), nil
		case missingCredsAddress:
			return []byte(credentials.NewErrCredentialsNotFound().Error()), errProgramExited
		case invalidServerAddress:
			return []byte("program failed"), errProgramExited
		case "":
			return []byte(credentials.NewErrCredentialsMissingServerURL().Error()), errProgramExited
		}
	case "store":
		var c credentials.Credentials
		err := json.NewDecoder(strings.NewReader(inS)).Decode(&c)
		if err != nil {
			return []byte("error storing credentials"), errProgramExited
		}
		switch c.ServerURL {
		case validServerAddress:
			return nil, nil
		case validServerAddress2:
			return nil, nil
		default:
			return []byte("error storing credentials"), errProgramExited
		}
	case "list":
		return []byte(fmt.Sprintf(`{"%s": "%s"}`, validServerAddress, validUsername)), nil

	}

	return []byte(fmt.Sprintf("unknown argument %q with %q", m.arg, inS)), errProgramExited
}

// Input sets the input to send to a remote credentials-helper.
func (m *mockProgram) Input(in io.Reader) {
	m.input = in
}

func mockProgramFn(args ...string) Program {
	return &mockProgram{
		arg: args[0],
	}
}

func ExampleStore() {
	p := NewShellProgramFunc("docker-credential-pass")

	c := &credentials.Credentials{
		ServerURL: "https://registry.example.com",
		Username:  "exampleuser",
		Secret:    "my super secret token",
	}

	if err := Store(p, c); err != nil {
		_, _ = fmt.Println(err)
	}
}

func TestStore(t *testing.T) {
	valid := []credentials.Credentials{
		{ServerURL: validServerAddress, Username: "foo", Secret: "bar"},
		{ServerURL: validServerAddress2, Username: "<token>", Secret: "abcd1234"},
	}

	for _, v := range valid {
		if err := Store(mockProgramFn, &v); err != nil {
			t.Error(err)
		}
	}

	invalid := []credentials.Credentials{
		{ServerURL: invalidServerAddress, Username: "foo", Secret: "bar"},
	}

	for _, v := range invalid {
		if err := Store(mockProgramFn, &v); err == nil {
			t.Errorf("Expected error for server %s, got nil", v.ServerURL)
		}
	}
}

func ExampleGet() {
	p := NewShellProgramFunc("docker-credential-pass")

	creds, err := Get(p, "https://registry.example.com")
	if err != nil {
		_, _ = fmt.Println(err)
	}

	_, _ = fmt.Printf("Got credentials for user `%s` in `%s`\n", creds.Username, creds.ServerURL)
}

func TestGet(t *testing.T) {
	valid := []credentials.Credentials{
		{ServerURL: validServerAddress, Username: "foo", Secret: "bar"},
		{ServerURL: validServerAddress2, Username: "<token>", Secret: "abcd1234"},
	}

	for _, v := range valid {
		c, err := Get(mockProgramFn, v.ServerURL)
		if err != nil {
			t.Fatal(err)
		}

		if c.Username != v.Username {
			t.Errorf("expected username `%s`, got %s", v.Username, c.Username)
		}
		if c.Secret != v.Secret {
			t.Errorf("expected secret `%s`, got %s", v.Secret, c.Secret)
		}
	}

	missingServerURLErr := credentials.NewErrCredentialsMissingServerURL()

	invalid := []struct {
		serverURL string
		err       string
	}{
		{
			serverURL: missingCredsAddress,
			err:       credentials.NewErrCredentialsNotFound().Error(),
		},
		{
			serverURL: invalidServerAddress,
			err:       "error getting credentials - err: exited 1, out: `program failed`",
		},
		{
			err: fmt.Sprintf("error getting credentials - err: %s, out: `%s`", missingServerURLErr.Error(), missingServerURLErr.Error()),
		},
	}

	for _, v := range invalid {
		_, err := Get(mockProgramFn, v.serverURL)
		if err == nil {
			t.Fatalf("Expected error for server %s, got nil", v.serverURL)
		}
		if err.Error() != v.err {
			t.Errorf("Expected error `%s`, got `%v`", v.err, err)
		}
	}
}

func ExampleErase() {
	p := NewShellProgramFunc("docker-credential-pass")

	if err := Erase(p, "https://registry.example.com"); err != nil {
		_, _ = fmt.Println(err)
	}
}

func TestErase(t *testing.T) {
	if err := Erase(mockProgramFn, validServerAddress); err != nil {
		t.Error(err)
	}

	if err := Erase(mockProgramFn, invalidServerAddress); err == nil {
		t.Errorf("Expected error for server %s, got nil", invalidServerAddress)
	}
}

func TestList(t *testing.T) {
	auths, err := List(mockProgramFn)
	if err != nil {
		t.Fatal(err)
	}

	if username, exists := auths[validServerAddress]; !exists || username != validUsername {
		t.Errorf("auths[%s] returned %s, %t; expected %s, %t", validServerAddress, username, exists, validUsername, true)
	}
}
