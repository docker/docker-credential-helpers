package wincred

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// maxCredentialBlobSize is CRED_MAX_CREDENTIAL_BLOB_SIZE (5 * 512 bytes).
	// The CredentialBlob field of a credential cannot be larger than this;
	// CredWrite fails for larger values, which surfaced as the unhelpful
	// "The stub received bad data." error.
	maxCredentialBlobSize = 5 * 512

	// maxCredentialAttributeValueSize is CRED_MAX_VALUE_SIZE (256 bytes), the
	// maximum size of the value of a credential attribute.
	maxCredentialAttributeValueSize = 256

	// maxCredentialAttributes is CRED_MAX_ATTRIBUTES, the maximum number of
	// attributes a credential can have, including the "label" attribute used
	// by this helper.
	maxCredentialAttributes = 64

	// secretChunkAttributePrefix is the prefix of the keywords of attributes
	// that hold the part of a secret that does not fit in the CredentialBlob.
	secretChunkAttributePrefix = "secret-chunk-"
)

// maxSecretSize is the largest secret a single credential can hold: the
// CredentialBlob, plus the credential attributes left after reserving one for
// the "label" attribute.
const maxSecretSize = maxCredentialBlobSize + (maxCredentialAttributes-1)*maxCredentialAttributeValueSize

// splitSecret splits secret into a credential blob and chunks to store in
// credential attributes. Secrets that fit in a credential blob are returned
// as-is, without chunks, to keep the storage format of credentials written by
// older versions of this helper unchanged.
func splitSecret(secret []byte) ([]byte, [][]byte, error) {
	if len(secret) > maxSecretSize {
		return nil, nil, fmt.Errorf("secret is %d bytes, but the Windows Credential Manager supports at most %d bytes per credential", len(secret), maxSecretSize)
	}
	if len(secret) <= maxCredentialBlobSize {
		return secret, nil, nil
	}

	blob := secret[:maxCredentialBlobSize]
	rest := secret[maxCredentialBlobSize:]
	chunks := make([][]byte, 0, (len(rest)+maxCredentialAttributeValueSize-1)/maxCredentialAttributeValueSize)
	for len(rest) > 0 {
		size := len(rest)
		if size > maxCredentialAttributeValueSize {
			size = maxCredentialAttributeValueSize
		}
		chunks = append(chunks, rest[:size])
		rest = rest[size:]
	}
	return blob, chunks, nil
}

// joinSecret reassembles a secret from a credential blob and the credential
// attributes whose keyword starts with secretChunkAttributePrefix. The attributes
// are expected to be indexed from zero, without gaps. Attributes with other
// keywords are ignored.
func joinSecret(blob []byte, attributes map[string][]byte) ([]byte, error) {
	chunks := make(map[int][]byte, len(attributes))
	maxIndex := -1
	for keyword, value := range attributes {
		if !strings.HasPrefix(keyword, secretChunkAttributePrefix) {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(keyword, secretChunkAttributePrefix))
		if err != nil || index < 0 || index >= maxCredentialAttributes {
			return nil, fmt.Errorf("credential has an invalid secret chunk attribute %q", keyword)
		}
		if _, ok := chunks[index]; ok {
			return nil, fmt.Errorf("credential has a duplicate secret chunk attribute %q", keyword)
		}
		chunks[index] = value
		if index > maxIndex {
			maxIndex = index
		}
	}
	if len(chunks) == 0 {
		return blob, nil
	}
	if len(blob) != maxCredentialBlobSize {
		return nil, fmt.Errorf("credential blob is %d bytes, expected %d bytes for a chunked secret", len(blob), maxCredentialBlobSize)
	}

	secret := make([]byte, 0, len(blob)+(maxIndex+1)*maxCredentialAttributeValueSize)
	secret = append(secret, blob...)
	for index := 0; index <= maxIndex; index++ {
		chunk, ok := chunks[index]
		if !ok {
			return nil, fmt.Errorf("credential is missing secret chunk %d of %d", index, maxIndex+1)
		}
		secret = append(secret, chunk...)
	}
	return secret, nil
}
