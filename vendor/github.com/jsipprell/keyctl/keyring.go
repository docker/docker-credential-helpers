// Copyright 2015 Jesse Sipprell. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

// A Go interface to linux kernel keyrings (keyctl interface)
package keyctl

// All Keys and Keyrings have unique 32-bit serial number identifiers.
type Id interface {
	Id() int32
	Info() (Info, error)

	private()
}

// Basic interface to a linux keyctl keyring.
type Keyring interface {
	Id
	Add(string, []byte) (*Key, error)
	Search(string) (*Key, error)
	SetDefaultTimeout(uint)
}

// Named keyrings are user-created keyrings linked to a parent keyring. The
// parent can be either named or one of the in-built keyrings (session, group
// etc). The in-built keyrings have no parents. Keyring searching is performed
// hierarchically.
type NamedKeyring interface {
	Keyring
	Name() string
}

type keyring struct {
	id         keyId
	defaultTtl uint
}

type namedKeyring struct {
	*keyring
	parent keyId
	name   string // for non-anonymous keyrings
	ttl    uint
}

func (kr *keyring) private() {}

// Returns the 32-bit kernel identifier of a keyring
func (kr *keyring) Id() int32 {
	return int32(kr.id)
}

// Returns information about a keyring.
func (kr *keyring) Info() (Info, error) {
	return getInfo(kr.id)
}

// Return the name of a NamedKeyring that was set when the keyring was created
// or opened.
func (kr *namedKeyring) Name() string {
	return kr.name
}

// Set a default timeout, in seconds, after which newly added keys will be
// destroyed.
func (kr *keyring) SetDefaultTimeout(nsecs uint) {
	kr.defaultTtl = nsecs
}

// Add a new key to a keyring. The key can be searched for later by name.
func (kr *keyring) Add(name string, key []byte) (*Key, error) {
	r, err := add_key("user", name, key, int32(kr.id))
	if err == nil {
		key := &Key{Name: name, id: keyId(r), ring: kr.id}
		if kr.defaultTtl != 0 {
			err = key.ExpireAfter(kr.defaultTtl)
		}
		return key, err
	}

	return nil, err
}

// Search for a key by name, this also searches child keyrings linked to this
// one. The key, if found, is linked to the top keyring that Search() was called
// from.
func (kr *keyring) Search(name string) (*Key, error) {
	id, err := searchKeyring(kr.id, name, "user")
	if err == nil {
		return &Key{Name: name, id: id, ring: kr.id}, nil
	}
	return nil, err
}

// Return the current login session keyring
func SessionKeyring() (Keyring, error) {
	return newKeyring(keySpecSessionKeyring)
}

// Return the current user-session keyring (part of session, but private to
// current user)
func UserSessionKeyring() (Keyring, error) {
	return newKeyring(keySpecUserSessionKeyring)
}

// Return the current group keyring.
func GroupKeyring() (Keyring, error) {
	return newKeyring(keySpecGroupKeyring)
}

// Return the keyring specific to the current executing thread.
func ThreadKeyring() (Keyring, error) {
	return newKeyring(keySpecThreadKeyring)
}

// Return the keyring specific to the current executing process.
func ProcessKeyring() (Keyring, error) {
	return newKeyring(keySpecProcessKeyring)
}

// Creates a new named-keyring linked to a parent keyring. The parent may be
// one of those returned by SessionKeyring(), UserSessionKeyring() and friends
// or it may be an existing named-keyring. When searching is performed, all
// keyrings form a hierarchy and are searched top-down. If the keyring already
// exists it will be destroyed and a new one with the same name created. Named
// sub-keyrings inherit their initial ttl (if set) from the parent but can
// outlive the parent as the timer is restarted at creation.
func CreateKeyring(parent Keyring, name string) (NamedKeyring, error) {
	var ttl uint

	parentId := keyId(parent.Id())
	kr, err := createKeyring(parentId, name)
	if err != nil {
		return nil, err
	}

	if pkr, ok := parent.(*namedKeyring); ok {
		ttl = pkr.ttl
	}
	ring := &namedKeyring{
		keyring: kr,
		parent:  parentId,
		name:    name,
		ttl:     ttl,
	}

	if ttl > 0 {
		err = keyctl_SetTimeout(ring.id, ttl)
	}

	return ring, nil
}

// Search for and open an existing keyring with the given name linked to a
// parent keyring (at any depth).
func OpenKeyring(parent Keyring, name string) (NamedKeyring, error) {
	parentId := keyId(parent.Id())
	id, err := searchKeyring(parentId, name, "keyring")
	if err != nil {
		return nil, err
	}

	return &namedKeyring{
		keyring: &keyring{id: id},
		parent:  parentId,
		name:    name,
	}, nil
}

// Set the time to live in seconds for an entire keyring and all of its keys.
// Only named keyrings can have their time-to-live set, the in-built keyrings
// cannot (Session, UserSession, etc).
func SetKeyringTTL(kr NamedKeyring, nsecs uint) error {
	err := keyctl_SetTimeout(keyId(kr.Id()), nsecs)
	if err == nil {
		kr.(*namedKeyring).ttl = nsecs
	}
	return err
}

// Unlink an object from a keyring
func Unlink(parent Keyring, child Id) error {
	return keyctl_Unlink(keyId(parent.Id()), keyId(child.Id()))
}

// Unlink a named keyring from its parent.
func UnlinkKeyring(kr NamedKeyring) error {
	return keyctl_Unlink(keyId(kr.Id()), kr.(*namedKeyring).parent)
}
