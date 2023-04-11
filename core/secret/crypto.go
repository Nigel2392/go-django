package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
)

type HashUint uint32

func (h HashUint) String() string {
	return fmt.Sprintf("%x", uint32(h))
}

func (h HashUint) Compare(other HashUint) bool {
	return h == other
}

// Hashes any string to a 32-bit integer
func FnvHash(s string) HashUint {
	h := fnv.New32a()
	h.Write([]byte(s))
	return HashUint(h.Sum32())
}

type _htmlSafe struct {
	k **secretKey
}

// Encrypt data with the secret key.
func (s *_htmlSafe) Encrypt(data string) (string, error) {
	var cipherText, err = (*s.k).bytes.Encrypt([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(cipherText), nil
}

// Decrypt data with the secret key.
func (s *_htmlSafe) Decrypt(data string) (string, error) {
	var cipherText, err = base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	plainData, err := (*s.k).bytes.Decrypt(cipherText)
	if err != nil {
		return "", err
	}
	return string(plainData), nil
}

type _byteCrypto struct {
	k **secretKey
}

// Encrypt data with the secret key.
func (s *_byteCrypto) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher((*s.k).KeyFromSecret()[:])
	if err != nil {
		return nil, err
	}

	cgm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	var nonce = make([]byte, cgm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	var cipherText = cgm.Seal(nonce, nonce, data, nil)
	return cipherText, nil
}

// Decrypt data with the secret key.
func (s *_byteCrypto) Decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher((*s.k).KeyFromSecret()[:])
	if err != nil {
		return nil, err
	}

	cgm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := cgm.NonceSize()
	if len(data) < nonceSize {
		return nil, err
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainData, err := cgm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plainData, nil
}
