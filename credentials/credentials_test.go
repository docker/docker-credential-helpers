package credentials

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type memoryStore struct {
	creds map[string]*Credentials
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		creds: make(map[string]*Credentials),
	}
}

func (m *memoryStore) Add(creds *Credentials) error {
	m.creds[creds.ServerURL] = creds
	return nil
}

func (m *memoryStore) Delete(serverURL string) error {
	delete(m.creds, serverURL)
	return nil
}

func (m *memoryStore) Get(serverURL string) (string, string, error) {
	c, ok := m.creds[serverURL]
	if !ok {
		return "", "", fmt.Errorf("creds not found for %s", serverURL)
	}
	return c.Username, c.Secret, nil
}

func TestStore(t *testing.T) {
	serverURL := "https://index.docker.io/v1/"
	creds := &Credentials{
		ServerURL: serverURL,
		Username:  "foo",
		Secret:    "bar",
	}
	b, err := json.Marshal(creds)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(b)

	h := newMemoryStore()
	if err := store(h, in); err != nil {
		t.Fatal(err)
	}

	c, ok := h.creds[serverURL]
	if !ok {
		t.Fatalf("creds not found for %s\n", serverURL)
	}

	if c.Username != "foo" {
		t.Fatalf("expected username foo, got %s\n", c.Username)
	}

	if c.Secret != "bar" {
		t.Fatalf("expected username bar, got %s\n", c.Secret)
	}
}

func TestGet(t *testing.T) {
	serverURL := "https://index.docker.io/v1/"
	creds := &Credentials{
		ServerURL: serverURL,
		Username:  "foo",
		Secret:    "bar",
	}
	b, err := json.Marshal(creds)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(b)

	h := newMemoryStore()
	if err := store(h, in); err != nil {
		t.Fatal(err)
	}

	buf := strings.NewReader(serverURL)
	w := new(bytes.Buffer)
	if err := get(h, buf, w); err != nil {
		t.Fatal(err)
	}

	if w.Len() == 0 {
		t.Fatalf("expected output in the writer, got %d", w.Len())
	}

	var c credentialsGetResponse
	if err := json.NewDecoder(w).Decode(&c); err != nil {
		t.Fatal(err)
	}

	if c.Username != "foo" {
		t.Fatalf("expected username foo, got %s\n", c.Username)
	}

	if c.Secret != "bar" {
		t.Fatalf("expected username bar, got %s\n", c.Secret)
	}
}

func TestErase(t *testing.T) {
	serverURL := "https://index.docker.io/v1/"
	creds := &Credentials{
		ServerURL: serverURL,
		Username:  "foo",
		Secret:    "bar",
	}
	b, err := json.Marshal(creds)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(b)

	h := newMemoryStore()
	if err := store(h, in); err != nil {
		t.Fatal(err)
	}

	buf := strings.NewReader(serverURL)
	if err := erase(h, buf); err != nil {
		t.Fatal(err)
	}

	w := new(bytes.Buffer)
	if err := get(h, buf, w); err == nil {
		t.Fatal("expected error getting missing creds, got empty")
	}
}
