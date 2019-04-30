package registryurl

import (
	"errors"
	"net/url"
	"strings"
)

// Parse parses and validates a given serverURL to an url.URL, and
// returns an error if validation failed. Querystring parameters are
// omitted in the resulting URL, because they are not used in the helper.
//
// If serverURL does not have a valid scheme, `//` is used as scheme
// before parsing. This prevents the hostname being used as path,
// and the credentials being stored without host.
func Parse(registryURL string) (*url.URL, error) {
	// Check if registryURL has a scheme, otherwise add `//` as scheme.
	if !strings.Contains(registryURL, "://") && !strings.HasPrefix(registryURL, "//") {
		registryURL = "//" + registryURL
	}

	u, err := url.Parse(registryURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "" && u.Scheme != "https" && u.Scheme != "http" {
		return nil, errors.New("unsupported scheme: " + u.Scheme)
	}

	if GetHostname(u) == "" {
		return nil, errors.New("no hostname in URL")
	}

	u.RawQuery = ""
	return u, nil
}
