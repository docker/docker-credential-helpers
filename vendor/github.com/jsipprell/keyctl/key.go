package keyctl

import (
	"time"
)

// Represents a single key linked to one or more kernel keyrings.
type Key struct {
	Name string

	id, ring keyId
	size     int
	ttl      time.Duration
}

func (k *Key) private() {}

// Returns the 32-bit kernel identifier for a specific key
func (k *Key) Id() int32 {
	return int32(k.id)
}

// To expire a key automatically after some period of time call this method.
func (k *Key) ExpireAfter(nsecs uint) error {
	k.ttl = time.Duration(nsecs) * time.Second

	return keyctl_SetTimeout(k.id, nsecs)
}

// Return information about a key.
func (k *Key) Info() (Info, error) {
	return getInfo(k.id)
}

// Get the key's value as a byte slice
func (k *Key) Get() ([]byte, error) {
	var (
		b        []byte
		err      error
		sizeRead int
	)

	if k.size == 0 {
		k.size = 512
	}

	size := k.size

	b = make([]byte, int(size))
	sizeRead = size + 1
	for sizeRead > size {
		r1, err := keyctl_Read(k.id, &b[0], size)
		if err != nil {
			return nil, err
		}

		if sizeRead = int(r1); sizeRead > size {
			b = make([]byte, sizeRead)
			size = sizeRead
			sizeRead = size + 1
		} else {
			k.size = sizeRead
		}
	}
	return b[:k.size], err
}

// Set the key's value from a bytes slice. Expiration, if active, is reset by calling this method.
func (k *Key) Set(b []byte) error {
	err := updateKey(k.id, b)
	if err == nil && k.ttl > 0 {
		err = k.ExpireAfter(uint(k.ttl.Seconds()))
	}
	return err
}

// Unlink a key from the keyring it was loaded from (or added to). If the key
// is not linked to any other keyrings, it is destroyed.
func (k *Key) Unlink() error {
	return keyctl_Unlink(k.id, k.ring)
}
