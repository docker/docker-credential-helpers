package osxkeychain

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Security -framework Foundation

#include "osxkeychain_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unsafe"

	"github.com/docker/docker-credential-helpers/credentials"
)

// errCredentialsNotFound is the specific error message returned by OS X
// when the credentials are not in the keychain.
const errCredentialsNotFound = "The specified item could not be found in the keychain."

type osxkeychain struct{}

// New creates a new osxkeychain.
func New() credentials.Helper {
	return osxkeychain{}
}

// Add adds new credentials to the keychain.
func (h osxkeychain) Add(creds *credentials.Credentials) error {
	s, err := splitServer(creds.ServerURL)
	if err != nil {
		return err
	}
	defer freeServer(s)

	username := C.CString(creds.Username)
	defer C.free(unsafe.Pointer(username))
	password := C.CString(creds.Password)
	defer C.free(unsafe.Pointer(password))

	errMsg := C.keychain_add(s, username, password)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		return errors.New(C.GoString(errMsg))
	}

	return nil
}

// Delete removes credentials from the keychain.
func (h osxkeychain) Delete(serverURL string) error {
	s, err := splitServer(serverURL)
	if err != nil {
		return err
	}
	defer freeServer(s)

	errMsg := C.keychain_delete(s)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		return errors.New(C.GoString(errMsg))
	}

	return nil
}

// Get returns the username and password to use for a given registry server URL.
func (h osxkeychain) Get(serverURL string) (string, string, error) {
	s, err := splitServer(serverURL)
	if err != nil {
		return "", "", err
	}
	defer freeServer(s)

	var usernameLen C.uint
	var username *C.char
	var passwordLen C.uint
	var password *C.char
	defer C.free(unsafe.Pointer(username))
	defer C.free(unsafe.Pointer(password))

	errMsg := C.keychain_get(s, &usernameLen, &username, &passwordLen, &password)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		goMsg := C.GoString(errMsg)

		if goMsg == errCredentialsNotFound {
			return "", "", credentials.ErrCredentialsNotFound
		}

		return "", "", errors.New(goMsg)
	}

	user := C.GoStringN(username, C.int(usernameLen))
	pass := C.GoStringN(password, C.int(passwordLen))
	return user, pass, nil
}

func splitServer(serverURL string) (*C.struct_Server, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	hostAndPort := strings.Split(u.Host, ":")
	host := hostAndPort[0]
	var port int
	if len(hostAndPort) == 2 {
		p, err := strconv.Atoi(hostAndPort[1])
		if err != nil {
			return nil, err
		}
		port = p
	}

	proto := C.kSecProtocolTypeHTTPS
	if u.Scheme != "https" {
		proto = C.kSecProtocolTypeHTTP
	}

	return &C.struct_Server{
		proto: C.SecProtocolType(proto),
		host:  C.CString(host),
		port:  C.uint(port),
		path:  C.CString(u.Path),
	}, nil
}

func freeServer(s *C.struct_Server) {
	C.free(unsafe.Pointer(s.host))
	C.free(unsafe.Pointer(s.path))
}
