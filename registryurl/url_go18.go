//+build go1.8

package registryurl

import (
	url "net/url"
)

// GetHostname returns the hostname of the URL
func GetHostname(u *url.URL) string {
	return u.Hostname()
}

// GetPort returns the port number of the URL
func GetPort(u *url.URL) string {
	return u.Port()
}
