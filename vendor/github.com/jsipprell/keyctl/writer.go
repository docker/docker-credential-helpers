package keyctl

import (
	"bytes"
	"errors"
	"io"
)

type Flusher interface {
	io.Writer
	io.Closer
	Flush() error
}

// Error returned when attempting to close or flush an already closed stream
var ErrStreamClosed = errors.New("keyctl write stream closed")

type writer struct {
	*bytes.Buffer
	key    Id
	name   string
	closed bool
}

// Close a stream writer. *This or Flush() MUST be called in order to flush
// the key value to the kernel.
func (w *writer) Close() error {
	if !w.closed {
		defer setClosed(w)
		return w.Flush()
	}
	return ErrStreamClosed
}

// Flush the current stream writer buffer key data to the kernel. New writes
// after this will need to be re-flushed or have Close() called.
func (w *writer) Flush() (err error) {
	if !w.closed {
		switch t := w.key.(type) {
		case Keyring:
			var key Id
			key, err = t.Add(w.name, w.Bytes())
			if err == nil {
				w.key = key
			}
		case *Key:
			err = updateKey(t.id, w.Bytes())
			if err == nil && t.ttl != 0 {
				err = t.ExpireAfter(uint(t.ttl.Seconds()))
			}
		}
		return
	}
	return ErrStreamClosed
}

func setClosed(w *writer) {
	w.closed = true
}

// Create a new stream writer to write key data to. The writer MUST Close() or
// Flush() the stream before the data will be flushed to the kernel.
func NewWriter(key *Key) Flusher {
	return &writer{Buffer: bytes.NewBuffer(make([]byte, 0, 1024)), key: key}
}

// Create a new key and stream writer with a given name on an open keyring.
func CreateWriter(name string, ring Keyring) (Flusher, error) {
	return &writer{Buffer: bytes.NewBuffer(make([]byte, 0, 1024)), key: ring, name: name}, nil
}
