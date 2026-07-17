package encryption

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"hash/fnv"
	"io"
)

const _KEY_ID_SIZE int = 4

type baseEncryption struct {
	secretId string
	secret   cipher.AEAD
	_m       map[string]cipher.AEAD
}

// fnv is fine here, this is just a fast path for when fallbackkeys get out of hand
func getKeyID(key []byte) []byte {
	h := fnv.New32a()
	h.Write(key)
	id := make([]byte, 4)
	binary.BigEndian.PutUint32(id, h.Sum32())
	return id
}

func newBaseEncryption(key []byte, fallbackKeys [][]byte) (Encryption, error) {
	var (
		fastPath = make(map[string]cipher.AEAD)
		secretId string
	)
	for i, k := range append([][]byte{key}, fallbackKeys...) {
		keyId := string(getKeyID(k))
		if i == 0 {
			secretId = keyId
		}

		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, ErrEncryption.WithCause(err)
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		fastPath[keyId] = gcm
	}

	return &baseEncryption{
		secretId: secretId,
		secret:   fastPath[secretId],
		_m:       fastPath,
	}, nil
}

func (b *baseEncryption) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {

	nonce := make([]byte, b.secret.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Create payload: [KeyID] [Nonce] [Ciphertext]
	out := make([]byte, _KEY_ID_SIZE+len(nonce)+len(plaintext)+b.secret.Overhead())
	copy(out[0:], b.secretId)
	copy(out[_KEY_ID_SIZE:], nonce)

	additionalData, _ := AdditionalDataFromContext(ctx)
	return b.secret.Seal(out[:_KEY_ID_SIZE+len(nonce)], nonce, plaintext, additionalData), nil
}

func (b *baseEncryption) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < _KEY_ID_SIZE+12 { // Minimum size: _KEY_ID_SIZE ID + 12 byte nonce
		return nil, ErrDecryption.Wrap("payload too short")
	}

	keyId := ciphertext[0:_KEY_ID_SIZE]
	gcm, ok := b._m[string(keyId)]
	if !ok {
		return nil, ErrDecryption.Wrap("key id not found")
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryption.Wrap("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[_KEY_ID_SIZE:_KEY_ID_SIZE+nonceSize], ciphertext[_KEY_ID_SIZE+nonceSize:]
	additionalData, _ := AdditionalDataFromContext(ctx)
	return gcm.Open(nil, nonce, ciphertext, additionalData)
}
