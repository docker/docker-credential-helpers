package osxkeychain

/*
#cgo CFLAGS: -x objective-c -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Security -framework Foundation -mmacosx-version-min=10.11

#include "osxkeychain_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"strconv"
	"unsafe"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/registryurl"

	"net/http"
	"net/url"
	"strings"
	"encoding/json"
	"io/ioutil"
	"os"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	IdToken string `json:"id_token"`
	ExpiresIn int `json:"expires_in"`
	TokenType string `json:"token_type"`
}

// errCredentialsNotFound is the specific error message returned by OS X
// when the credentials are not in the keychain.
const errCredentialsNotFound = "The specified item could not be found in the keychain."

// errCredentialsNotFound is the specific error message returned by OS X
// when environment does not allow showing dialog to unlock keychain.
const errInteractionNotAllowed = "User interaction is not allowed."

// ErrInteractionNotAllowed is returned if keychain password prompt can not be shown.
var ErrInteractionNotAllowed = errors.New(`keychain cannot be accessed because the current session does not allow user interaction. The keychain may be locked; unlock it by running "security -v unlock-keychain ~/Library/Keychains/login.keychain-db" and try again`)

// Osxkeychain handles secrets using the OS X Keychain as store.
type Osxkeychain struct{}

// Add adds new credentials to the keychain.
func (h Osxkeychain) Add(creds *credentials.Credentials) error {
	h.Delete(creds.ServerURL)

	s, err := splitServer(creds.ServerURL)
	if err != nil {
		return err
	}
	defer freeServer(s)

	label := C.CString(credentials.CredsLabel)
	defer C.free(unsafe.Pointer(label))
	username := C.CString(creds.Username)
	defer C.free(unsafe.Pointer(username))
	secret := C.CString(creds.Secret)
	defer C.free(unsafe.Pointer(secret))

	errMsg := C.keychain_add(s, label, username, secret)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		return errors.New(C.GoString(errMsg))
	}

	return nil
}

// Delete removes credentials from the keychain.
func (h Osxkeychain) Delete(serverURL string) error {
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

// Get returns the username and secret to use for a given registry server URL.
func (h Osxkeychain) Get(serverURL string) (string, string, error) {
	s, err := splitServer(serverURL)
	if err != nil {
		return "", "", err
	}
	defer freeServer(s)
	
	clientId, okClientId := os.LookupEnv("CLIENT_ID")

	if !okClientId {
		return "", "", errors.New("env variable CLIENT_ID is not found")
	}

	clientSecret, okClientSecret := os.LookupEnv("CLIENT_SECRET")

	if !okClientSecret {
		return "", "", errors.New("env variable CLIENT_SECRET is not found")
	}
	
	auth, err := GetAuthorizationToken(clientId, clientSecret)

	return "muniker", auth.AccessToken, nil
}

// Get access token from amazoncognito using client credentials.
func GetAuthorizationToken(clientId string, clientSecret string) (*AuthResponse, error) {	
	var cognitoAuthEndpoint = "https://azad.auth.us-west-2.amazoncognito.com/oauth2/token"

	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "client_credentials")
	encodedData := data.Encode()

	response, httpErr := http.Post(cognitoAuthEndpoint, "application/x-www-form-urlencoded", strings.NewReader(encodedData))
	
	if httpErr != nil {
		return nil, httpErr
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body) 

	var authResponse AuthResponse
	
	unmarshalErr := json.Unmarshal(body, &authResponse)
	if unmarshalErr != nil {
		return nil, httpErr
    }

	return &authResponse, nil
}

// List returns the stored URLs and corresponding usernames.
func (h Osxkeychain) List() (map[string]string, error) {
	credsLabelC := C.CString(credentials.CredsLabel)
	defer C.free(unsafe.Pointer(credsLabelC))

	var pathsC **C.char
	defer C.free(unsafe.Pointer(pathsC))
	var acctsC **C.char
	defer C.free(unsafe.Pointer(acctsC))
	var listLenC C.uint
	errMsg := C.keychain_list(credsLabelC, &pathsC, &acctsC, &listLenC)
	defer C.freeListData(&pathsC, listLenC)
	defer C.freeListData(&acctsC, listLenC)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		goMsg := C.GoString(errMsg)
		if goMsg == errCredentialsNotFound {
			return make(map[string]string), nil
		}
		if goMsg == errInteractionNotAllowed {
			return nil, ErrInteractionNotAllowed
		}

		return nil, errors.New(goMsg)
	}

	var listLen int
	listLen = int(listLenC)
	pathTmp := (*[1 << 30]*C.char)(unsafe.Pointer(pathsC))[:listLen:listLen]
	acctTmp := (*[1 << 30]*C.char)(unsafe.Pointer(acctsC))[:listLen:listLen]
	// taking the array of c strings into go while ignoring all the stuff irrelevant to credentials-helper
	resp := make(map[string]string)
	for i := 0; i < listLen; i++ {
		if C.GoString(pathTmp[i]) == "0" {
			continue
		}
		resp[C.GoString(pathTmp[i])] = C.GoString(acctTmp[i])
	}
	return resp, nil
}

func splitServer(serverURL string) (*C.struct_Server, error) {
	u, err := registryurl.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	proto := C.kSecProtocolTypeHTTPS
	if u.Scheme == "http" {
		proto = C.kSecProtocolTypeHTTP
	}
	var port int
	p := u.Port()
	if p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
	}

	return &C.struct_Server{
		proto: C.SecProtocolType(proto),
		host:  C.CString(u.Hostname()),
		port:  C.uint(port),
		path:  C.CString(u.Path),
	}, nil
}

func freeServer(s *C.struct_Server) {
	C.free(unsafe.Pointer(s.host))
	C.free(unsafe.Pointer(s.path))
}
