// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 sudoforge <sudoforge.com>

// Package gopass implements a `gopass` credential helper. Passwords are
// stored as arguments to gopass of the form:
//
//	<NAMESPACE>/<b64-encoded-url>/<b64-encoded-username>
//
// We base64-url encode the registry URL (using base64.URLEncoding), because
// under the hood gopass uses files and folders. Not encoding the values would
// cause forward slashes in the URL to nest the entry under additional prefixes
// ("folders").
//
// We encode the username for much the same reason, and use it as part of the
// entry's name in order to support a potential future in which
// docker-credential-helpers is refactored to support multiple registry
// accounts.
//
// N.B.: This helper is backwards-compatible with the `pass` helper as of
// docker/docker-credential-helpers:f9d3010165b642df37215b1be945552f2c6f0e3b.
package gopass

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/gopasspw/gopass/pkg/gopass/api"
	"github.com/gopasspw/gopass/pkg/gopass/secrets"
)

// NAMESPACE is the prefix for all entries this helper manages
const NAMESPACE = "io.container.registries"

// Handler wraps an initialized gopass
type Handler struct {
	gp  *api.Gopass
	ctx context.Context
}

// New initializes a Handler or errors
func New() (*Handler, error) {
	ctx := context.Background()
	gp, err := api.New(ctx)
	if err != nil {
		return nil, err
	}

	helper := &Handler{
		gp:  gp,
		ctx: context.Background(),
	}

	return helper, nil
}

// Add creates a new entry in the password store
func (h *Handler) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	if creds.Username == "" {
		return errors.New("missing username")
	}

	name := path.Join(
		NAMESPACE,
		base64.URLEncoding.EncodeToString([]byte(creds.ServerURL)),
		base64.URLEncoding.EncodeToString([]byte(creds.Username)),
	)

	secret := secrets.New()
	secret.SetPassword(creds.Secret)

	if err := h.gp.Set(h.ctx, name, secret); err != nil {
		return fmt.Errorf("gopass error: error writing to store: %s", err)
	}

	return nil
}

// Delete removes ALL credentials for a given url
func (h *Handler) Delete(url string) error {
	if url == "" {
		return errors.New("missing registry url")
	}

	prefix := path.Join(
		NAMESPACE,
		base64.URLEncoding.EncodeToString([]byte(url)),
	)

	if err := h.gp.RemoveAll(h.ctx, prefix); err != nil {
		return fmt.Errorf("gopass error: %s", err)
	}

	return nil
}

func (h *Handler) listByPrefix(prefix string) ([]string, error) {
	entries, err := h.gp.List(h.ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to list entries by prefix '%s': %s", prefix, err)
	}

	var matches []string
	for _, e := range entries {
		if strings.HasPrefix(e, prefix) {
			matches = append(matches, e)
		}
	}

	return matches, nil
}

// Get returns the username and secret to use for a given registry server URL.
//
// Due to limitations in how Get is called, it cannot differentiate between
// multiple accounts for a given registry, and thus returns an error if there
// is not exactly one entry for the given registry url.
func (h *Handler) Get(url string) (string, string, error) {
	if url == "" {
		return "", "", errors.New("missing registry url")
	}

	prefix := path.Join(
		NAMESPACE,
		base64.URLEncoding.EncodeToString([]byte(url)),
	)

	matches, err := h.listByPrefix(prefix)
	if err != nil {
		return "", "", err
	}

	count := len(matches)
	if count < 1 {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	// we can only return one username at the moment (due to limitations in
	// docker-credential-helpers), so, return an error if we have more than one
	if count > 1 {
		return "", "", fmt.Errorf("found more than one entry for registry: %s", url)
	}

	parts := strings.Split(matches[0], "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("unexpected number of parts in secret '%s': %s", matches[0], err)
	}

	username, err := base64.URLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", fmt.Errorf("error decoding username from secret '%s': %s", matches[0], err)
	}

	secret, err := h.gp.Get(h.ctx, matches[0], "latest")
	if err != nil {
		return "", "", fmt.Errorf("unable to get secret '%s': %s", matches[0], err)
	}

	password := secret.Password()
	if password == "" {
		return "", "", fmt.Errorf("received an empty password for secret: %s", matches[0])
	}

	return string(username), password, nil
}

// List returns a map of registry URLs to usernames
func (h *Handler) List() (map[string]string, error) {
	matches, err := h.listByPrefix(NAMESPACE)
	if err != nil {
		return nil, err
	}

	res := map[string]string{}
	for _, m := range matches {
		parts := strings.Split(m, "/")
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected number of parts for entry: %s", m)
		}

		var url []byte
		if url, err = base64.URLEncoding.DecodeString(parts[1]); err != nil {
			return nil, fmt.Errorf("error decoding url from secret '%s': %s", m, err)
		}

		username := parts[2]
		if val, err := base64.URLEncoding.DecodeString(username); err == nil {
			username = string(val)
		}

		// due to limitations in the expected output format of this method, we
		// cannot support multiple registries, so we error out if we've already
		// stored this registry.
		if _, ok := res[string(url)]; ok {
			return nil, fmt.Errorf("encountered more than one entry for registry: %s", url)
		}

		res[string(url)] = username
	}

	return res, nil
}
