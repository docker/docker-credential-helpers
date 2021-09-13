package keyctl

import (
	"bytes"
	"io"
	"sync"
)

type reader struct {
	*bytes.Buffer
	key  *Key
	err  error
	once sync.Once
}

func (r *reader) Read(b []byte) (int, error) {
	r.once.Do(func() {
		buf, err := r.key.Get()
		if err != nil {
			r.err = err
		} else {
			r.Buffer = bytes.NewBuffer(buf)
		}
	})
	if r.err != nil {
		return -1, r.err
	}

	return r.Buffer.Read(b)
}

// Returns an io.Reader interface object which will read the key's data from
// the kernel.
func NewReader(key *Key) io.Reader {
	return &reader{key: key}
}

// Open an existing key on a keyring given its name
func OpenReader(name string, ring Keyring) (io.Reader, error) {
	key, err := ring.Search(name)
	if err == nil {
		return NewReader(key), nil
	}
	return nil, err
}
