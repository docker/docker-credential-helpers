package wincred

import (
	"strconv"
	"strings"
	"testing"
)

// testSecret returns a deterministic secret of the given size, with a distinct
// byte per position so that ordering errors are detected.
func testSecret(size int) []byte {
	secret := make([]byte, size)
	for i := range secret {
		secret[i] = byte('!' + (i*7)%90)
	}
	return secret
}

// chunkAttributes builds the attributes for the given chunks, the same way
// Wincred.Add does.
func chunkAttributes(chunks [][]byte) map[string][]byte {
	return chunkAttributesFrom(chunks, 0)
}

// chunkAttributesFrom builds the attributes for the given chunks, starting at
// the given index.
func chunkAttributesFrom(chunks [][]byte, offset int) map[string][]byte {
	attributes := make(map[string][]byte, len(chunks))
	for index, chunk := range chunks {
		attributes[secretChunkAttributePrefix+strconv.Itoa(offset+index)] = chunk
	}
	return attributes
}

func TestSplitSecret(t *testing.T) {
	tests := []struct {
		doc      string
		size     int
		chunks   int
		blobSize int
		wantErr  bool
	}{
		{doc: "empty secret", size: 0, chunks: 0, blobSize: 0},
		{doc: "short secret", size: 42, chunks: 0, blobSize: 42},
		{doc: "secret exactly filling the credential blob", size: maxCredentialBlobSize, chunks: 0, blobSize: maxCredentialBlobSize},
		{doc: "one byte over the credential blob", size: maxCredentialBlobSize + 1, chunks: 1, blobSize: maxCredentialBlobSize},
		{doc: "secret filling the credential blob and one attribute", size: maxCredentialBlobSize + maxCredentialAttributeValueSize, chunks: 1, blobSize: maxCredentialBlobSize},
		{doc: "secret filling the credential blob and spilling into a second attribute", size: maxCredentialBlobSize + maxCredentialAttributeValueSize + 1, chunks: 2, blobSize: maxCredentialBlobSize},
		{doc: "maximum secret size", size: maxSecretSize, chunks: maxCredentialAttributes - 1, blobSize: maxCredentialBlobSize},
		{doc: "secret too large", size: maxSecretSize + 1, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			secret := testSecret(tc.size)
			blob, chunks, err := splitSecret(secret)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(blob) != tc.blobSize {
				t.Errorf("blob is %d bytes, expected %d", len(blob), tc.blobSize)
			}
			if len(chunks) != tc.chunks {
				t.Fatalf("got %d chunks, expected %d", len(chunks), tc.chunks)
			}
			for i, chunk := range chunks {
				if len(chunk) > maxCredentialAttributeValueSize {
					t.Errorf("chunk %d is %d bytes, which exceeds the attribute limit of %d", i, len(chunk), maxCredentialAttributeValueSize)
				}
				if len(chunk) == 0 {
					t.Errorf("chunk %d is empty", i)
				}
			}
			// The split parts must reconstruct the original secret in order.
			reassembled := append([]byte{}, blob...)
			for _, chunk := range chunks {
				reassembled = append(reassembled, chunk...)
			}
			if string(reassembled) != string(secret) {
				t.Error("split parts do not reconstruct the original secret")
			}
		})
	}
}

func TestJoinSecretRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, maxCredentialBlobSize, maxCredentialBlobSize + 1, maxSecretSize - 1, maxSecretSize} {
		secret := testSecret(size)
		blob, chunks, err := splitSecret(secret)
		if err != nil {
			t.Fatalf("size %d: unexpected error: %v", size, err)
		}
		joined, err := joinSecret(blob, chunkAttributes(chunks))
		if err != nil {
			t.Fatalf("size %d: unexpected error: %v", size, err)
		}
		if string(joined) != string(secret) {
			t.Errorf("size %d: joined secret differs from the original", size)
		}
	}
}

func TestJoinSecretErrors(t *testing.T) {
	secret := testSecret(maxCredentialBlobSize + 3*maxCredentialAttributeValueSize)
	blob, chunks, err := splitSecret(secret)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		doc        string
		blob       []byte
		attributes map[string][]byte
	}{
		{
			doc:        "missing first chunk",
			blob:       blob,
			attributes: chunkAttributesFrom(chunks[1:], 1),
		},
		{
			doc:  "gap in chunks",
			blob: blob,
			attributes: map[string][]byte{
				secretChunkAttributePrefix + "0": chunks[0],
				secretChunkAttributePrefix + "2": chunks[2],
			},
		},
		{
			doc:  "duplicate chunk index",
			blob: blob,
			attributes: map[string][]byte{
				secretChunkAttributePrefix + "0":  chunks[0],
				secretChunkAttributePrefix + "1":  chunks[1],
				secretChunkAttributePrefix + "01": chunks[1],
			},
		},
		{
			doc:  "invalid chunk index",
			blob: blob,
			attributes: map[string][]byte{
				secretChunkAttributePrefix + "zero": chunks[0],
			},
		},
		{
			doc:  "negative chunk index",
			blob: blob,
			attributes: map[string][]byte{
				secretChunkAttributePrefix + "-1": chunks[0],
			},
		},
		{
			doc:        "chunked secret with a short blob",
			blob:       blob[:len(blob)-1],
			attributes: chunkAttributes(chunks),
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			if _, err := joinSecret(tc.blob, tc.attributes); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestJoinSecretIgnoresUnrelatedAttributes(t *testing.T) {
	blob, chunks, err := splitSecret(testSecret(maxCredentialBlobSize + 1))
	if err != nil {
		t.Fatal(err)
	}
	attributes := chunkAttributes(chunks)
	attributes["label"] = []byte("docker-credential-helpers")
	attributes["unrelated"] = []byte(strings.Repeat("x", 10))

	joined, err := joinSecret(blob, attributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(joined) != maxCredentialBlobSize+1 {
		t.Errorf("joined secret is %d bytes, expected %d", len(joined), maxCredentialBlobSize+1)
	}
}
