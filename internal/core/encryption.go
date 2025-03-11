package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type JWTEncryption struct{}

func (encryption *JWTEncryption) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aesgcm.Seal(nonce, nonce, plaintext, nil), nil
}
func (encryption *JWTEncryption) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	return aesgcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
}

var JWTEncrypter *JWTEncryption
