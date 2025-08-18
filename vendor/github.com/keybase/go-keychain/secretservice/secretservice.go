package secretservice

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	dbus "github.com/keybase/dbus"
)

// SecretServiceInterface
const SecretServiceInterface = "org.freedesktop.secrets"

// SecretServiceObjectPath
const SecretServiceObjectPath dbus.ObjectPath = "/org/freedesktop/secrets"

// DefaultCollection need not necessarily exist in the user's keyring.
const DefaultCollection dbus.ObjectPath = "/org/freedesktop/secrets/aliases/default"

// AuthenticationMode
type AuthenticationMode string

// AuthenticationInsecurePlain
const AuthenticationInsecurePlain AuthenticationMode = "plain"

// AuthenticationDHAES
const AuthenticationDHAES AuthenticationMode = "dh-ietf1024-sha256-aes128-cbc-pkcs7"

// NilFlags
const NilFlags = 0

// Attributes
type Attributes map[string]string

// Secret
type Secret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

// PromptCompletedResult
type PromptCompletedResult struct {
	Dismissed bool
	Paths     dbus.Variant
}

// SecretService
type SecretService struct {
	conn               *dbus.Conn
	signalCh           <-chan *dbus.Signal
	sessionOpenTimeout time.Duration
}

// Session
type Session struct {
	Mode    AuthenticationMode
	Path    dbus.ObjectPath
	Public  *big.Int
	Private *big.Int
	AESKey  []byte
}

// DefaultSessionOpenTimeout
const DefaultSessionOpenTimeout = 10 * time.Second

// NewService
func NewService() (*SecretService, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to open dbus connection: %w", err)
	}
	signalCh := make(chan *dbus.Signal, 16)
	conn.Signal(signalCh)
	_ = conn.AddMatchSignal(dbus.WithMatchOption("org.freedesktop.Secret.Prompt", "Completed"))
	return &SecretService{conn: conn, signalCh: signalCh, sessionOpenTimeout: DefaultSessionOpenTimeout}, nil
}

// SetSessionOpenTimeout
func (s *SecretService) SetSessionOpenTimeout(d time.Duration) {
	s.sessionOpenTimeout = d
}

// ServiceObj
func (s *SecretService) ServiceObj() dbus.BusObject {
	return s.conn.Object(SecretServiceInterface, SecretServiceObjectPath)
}

// Obj
func (s *SecretService) Obj(path dbus.ObjectPath) dbus.BusObject {
	return s.conn.Object(SecretServiceInterface, path)
}

type sessionOpenResponse struct {
	algorithmOutput dbus.Variant
	path            dbus.ObjectPath
}

func (s *SecretService) openSessionRaw(mode AuthenticationMode, sessionAlgorithmInput dbus.Variant) (resp sessionOpenResponse, err error) {
	err = s.ServiceObj().
		Call("org.freedesktop.Secret.Service.OpenSession", NilFlags, mode, sessionAlgorithmInput).
		Store(&resp.algorithmOutput, &resp.path)
	if err != nil {
		return sessionOpenResponse{}, fmt.Errorf("failed to open secretservice session: %w", err)
	}
	return resp, nil
}

// OpenSession
func (s *SecretService) OpenSession(mode AuthenticationMode) (session *Session, err error) {
	var sessionAlgorithmInput dbus.Variant

	session = new(Session)

	session.Mode = mode

	switch mode {
	case AuthenticationInsecurePlain:
		sessionAlgorithmInput = dbus.MakeVariant("")
	case AuthenticationDHAES:
		group := rfc2409SecondOakleyGroup()
		private, public, err := group.NewKeypair()
		if err != nil {
			return nil, err
		}
		session.Private = private
		session.Public = public
		sessionAlgorithmInput = dbus.MakeVariant(public.Bytes()) // math/big.Int.Bytes is big endian
	default:
		return nil, fmt.Errorf("unknown authentication mode %v", mode)
	}

	sessionOpenCh := make(chan sessionOpenResponse)
	errCh := make(chan error)
	go func() {
		resp, err := s.openSessionRaw(mode, sessionAlgorithmInput)
		if err != nil {
			errCh <- err
		} else {
			sessionOpenCh <- resp
		}
	}()

	var sessionAlgorithmOutput dbus.Variant
	// NOTE: If the timeout case is reached, the above goroutine is leaked.
	// This is not terrible because D-Bus calls have an internal 2-mintue
	// timeout, so the goroutine will finish eventually. If two OpenSessions
	// are called at the saime time, they'll be on different channels so
	// they won't interfere with each other.
	select {
	case resp := <-sessionOpenCh:
		sessionAlgorithmOutput = resp.algorithmOutput
		session.Path = resp.path
	case err := <-errCh:
		return nil, err
	case <-time.After(s.sessionOpenTimeout):
		return nil, fmt.Errorf("timed out after %s", s.sessionOpenTimeout)
	}

	switch mode {
	case AuthenticationInsecurePlain:
	case AuthenticationDHAES:
		theirPublicBigEndian, ok := sessionAlgorithmOutput.Value().([]byte)
		if !ok {
			return nil, errors.New("failed to coerce algorithm output value to byteslice")
		}
		group := rfc2409SecondOakleyGroup()
		theirPublic := new(big.Int)
		theirPublic.SetBytes(theirPublicBigEndian)
		aesKey, err := group.keygenHKDFSHA256AES128(theirPublic, session.Private)
		if err != nil {
			return nil, err
		}
		session.AESKey = aesKey
	default:
		return nil, fmt.Errorf("unknown authentication mode %v", mode)
	}

	return session, nil
}

// CloseSession
func (s *SecretService) CloseSession(session *Session) {
	s.Obj(session.Path).Call("org.freedesktop.Secret.Session.Close", NilFlags)
}

// SearchCollection
func (s *SecretService) SearchCollection(collection dbus.ObjectPath, attributes Attributes) (items []dbus.ObjectPath, err error) {
	err = s.Obj(collection).
		Call("org.freedesktop.Secret.Collection.SearchItems", NilFlags, attributes).
		Store(&items)
	if err != nil {
		return nil, fmt.Errorf("failed to search collection: %w", err)
	}
	return items, nil
}

// ReplaceBehavior
type ReplaceBehavior int

// ReplaceBehaviorDoNotReplace
const ReplaceBehaviorDoNotReplace = 0

// ReplaceBehaviorReplace
const ReplaceBehaviorReplace = 1

// CreateItem
func (s *SecretService) CreateItem(collection dbus.ObjectPath, properties map[string]dbus.Variant, secret Secret, replaceBehavior ReplaceBehavior) (item dbus.ObjectPath, err error) {
	var replace bool
	switch replaceBehavior {
	case ReplaceBehaviorDoNotReplace:
		replace = false
	case ReplaceBehaviorReplace:
		replace = true
	default:
		return "", fmt.Errorf("unknown replace behavior %d", replaceBehavior)
	}

	var prompt dbus.ObjectPath
	err = s.Obj(collection).
		Call("org.freedesktop.Secret.Collection.CreateItem", NilFlags, properties, secret, replace).
		Store(&item, &prompt)
	if err != nil {
		return "", fmt.Errorf("failed to create item: %w", err)
	}
	_, err = s.PromptAndWait(prompt)
	if err != nil {
		return "", err
	}
	return item, nil
}

// DeleteItem
func (s *SecretService) DeleteItem(item dbus.ObjectPath) (err error) {
	var prompt dbus.ObjectPath
	err = s.Obj(item).
		Call("org.freedesktop.Secret.Item.Delete", NilFlags).
		Store(&prompt)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	_, err = s.PromptAndWait(prompt)
	if err != nil {
		return err
	}
	return nil
}

// GetAttributes
func (s *SecretService) GetAttributes(item dbus.ObjectPath) (attributes Attributes, err error) {
	attributesV, err := s.Obj(item).GetProperty("org.freedesktop.Secret.Item.Attributes")
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}
	attributesMap, ok := attributesV.Value().(map[string]string)
	if !ok {
		return nil, errors.New("failed to coerce item attributes")
	}
	return Attributes(attributesMap), nil
}

// GetSecret
func (s *SecretService) GetSecret(item dbus.ObjectPath, session Session) (secretPlaintext []byte, err error) {
	var secretI []interface{}
	err = s.Obj(item).
		Call("org.freedesktop.Secret.Item.GetSecret", NilFlags, session.Path).
		Store(&secretI)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	secret := new(Secret)
	err = dbus.Store(secretI, &secret.Session, &secret.Parameters, &secret.Value, &secret.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal get secret result: %w", err)
	}

	switch session.Mode {
	case AuthenticationInsecurePlain:
		secretPlaintext = secret.Value
	case AuthenticationDHAES:
		plaintext, err := unauthenticatedAESCBCDecrypt(secret.Parameters, secret.Value, session.AESKey)
		if err != nil {
			return nil, nil
		}
		secretPlaintext = plaintext
	default:
		return nil, fmt.Errorf("cannot make secret for authentication mode %v", session.Mode)
	}

	return secretPlaintext, nil
}

// NullPrompt
const NullPrompt = "/"

// Unlock
func (s *SecretService) Unlock(items []dbus.ObjectPath) (err error) {
	var dummy []dbus.ObjectPath
	var prompt dbus.ObjectPath
	err = s.ServiceObj().
		Call("org.freedesktop.Secret.Service.Unlock", NilFlags, items).
		Store(&dummy, &prompt)
	if err != nil {
		return fmt.Errorf("failed to unlock items: %w", err)
	}
	_, err = s.PromptAndWait(prompt)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}
	return nil
}

// LockItems
func (s *SecretService) LockItems(items []dbus.ObjectPath) (err error) {
	var dummy []dbus.ObjectPath
	var prompt dbus.ObjectPath
	err = s.ServiceObj().
		Call("org.freedesktop.Secret.Service.Lock", NilFlags, items).
		Store(&dummy, &prompt)
	if err != nil {
		return fmt.Errorf("failed to lock items: %w", err)
	}
	_, err = s.PromptAndWait(prompt)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}
	return nil
}

// PromptDismissedError
type PromptDismissedError struct {
	err error
}

// Error
func (p PromptDismissedError) Error() string {
	return p.err.Error()
}

// PromptAndWait is NOT thread-safe.
func (s *SecretService) PromptAndWait(prompt dbus.ObjectPath) (paths *dbus.Variant, err error) {
	if prompt == NullPrompt {
		return nil, nil
	}
	call := s.Obj(prompt).Call("org.freedesktop.Secret.Prompt.Prompt", NilFlags, "Keyring Prompt")
	if call.Err != nil {
		return nil, fmt.Errorf("failed to prompt: %w", call.Err)
	}
	for {
		var result PromptCompletedResult
		select {
		case signal, ok := <-s.signalCh:
			if !ok {
				return nil, errors.New("prompt channel closed")
			}
			if signal == nil {
				continue
			}
			if signal.Name != "org.freedesktop.Secret.Prompt.Completed" {
				continue
			}
			err = dbus.Store(signal.Body, &result.Dismissed, &result.Paths)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal prompt result: %w", err)
			}
			if result.Dismissed {
				return nil, PromptDismissedError{errors.New("prompt dismissed")}
			}
			return &result.Paths, nil
		case <-time.After(30 * time.Second):
			return nil, errors.New("prompt timed out")
		}
	}
}

// NewSecretProperties
func NewSecretProperties(label string, attributes map[string]string) map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label":      dbus.MakeVariant(label),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(attributes),
	}
}

// NewSecret
func (session *Session) NewSecret(secretBytes []byte) (Secret, error) {
	switch session.Mode {
	case AuthenticationInsecurePlain:
		return Secret{
			Session:     session.Path,
			Parameters:  nil,
			Value:       secretBytes,
			ContentType: "application/octet-stream",
		}, nil
	case AuthenticationDHAES:
		iv, ciphertext, err := unauthenticatedAESCBCEncrypt(secretBytes, session.AESKey)
		if err != nil {
			return Secret{}, err
		}
		return Secret{
			Session:     session.Path,
			Parameters:  iv,
			Value:       ciphertext,
			ContentType: "application/octet-stream",
		}, nil
	default:
		return Secret{}, fmt.Errorf("cannot make secret for authentication mode %v", session.Mode)
	}
}
