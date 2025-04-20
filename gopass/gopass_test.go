// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 sudoforge <sudoforge.com>

package gopass

import (
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestGopassHelper(t *testing.T) {
	helper, err := New()
	if err != nil {
		t.Fatalf("unable to use helper 'gopass': %v", err)
	}

	creds := &credentials.Credentials{
		ServerURL: "https://gopass.docker.io:2376/v1",
		Username:  "gopass-username",
		Secret:    "gopass-password",
	}

	helper.Add(creds)

	creds.ServerURL = "https://gopass.docker.io:9999/v2"
	helper.Add(creds)

	credsList, err := helper.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(credsList) == 0 {
		t.Fatal("missing credentials from store")
	}

	for server, username := range credsList {

		if !(strings.Contains(server, "2376") ||
			strings.Contains(server, "9999")) {
			t.Fatalf("invalid url: %s", creds.ServerURL)
		}

		if username != "gopass-username" {
			t.Fatalf("invalid username: %v", username)
		}

		u, s, err := helper.Get(server)
		if err != nil {
			t.Fatal(err)
		}

		if u != username {
			t.Fatalf("invalid username %s", u)
		}

		if s != "gopass-password" {
			t.Fatalf("invalid secret: %s", s)
		}

		err = helper.Delete(server)
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = helper.Get(server)
		if !credentials.IsErrCredentialsNotFound(err) {
			t.Fatalf("expected credentials not found, actual: %v", err)
		}
	}

	credsList, err = helper.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(credsList) != 0 {
		t.Fatal("didn't delete all creds?")
	}
}

func TestMissingCred(t *testing.T) {
	helper, err := New()
	if err != nil {
		t.Fatalf("unable to use helper 'gopass': %v", err)
	}

	_, _, err = helper.Get("garbage")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected credentials not found, actual: %v", err)
	}
}
