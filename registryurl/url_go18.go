//+build go1.8

package registryurl

import (
	url "net/url"
)

func GetHostname(u *url.URL) string {
	return u.Hostname()
}

func GetPort(u *url.URL) string {
	return u.Port()
}
