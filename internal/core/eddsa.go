package core

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
)

type EdDSA struct {
	AccessPublicKey   any
	AccessPrivateKey  any
	RefreshPublicKey  any
	RefreshPrivateKey any
}

func LoadKeys() error {
	accessPrivateKeyPath := filepath.Join("secrets", "AccessPrivateKey.pem")
	accessPublicKeyPath := filepath.Join("secrets", "AccessPublicKey.pem")
	refreshPrivateKeyPath := filepath.Join("secrets", "RefreshPrivateKey.pem")
	refreshPublicKeyPath := filepath.Join("secrets", "RefreshPublicKey.pem")

	accessPrivateKey, err := loadKey(accessPrivateKeyPath, true)
	if err != nil {
		return err
	}

	accessPublicKey, err := loadKey(accessPublicKeyPath, false)
	if err != nil {
		return err
	}

	refreshPrivateKey, err := loadKey(refreshPrivateKeyPath, true)
	if err != nil {
		return err
	}

	refreshPublicKey, err := loadKey(refreshPublicKeyPath, false)
	if err != nil {
		return err
	}

	Ed25519Keys.AccessPublicKey = accessPublicKey
	Ed25519Keys.AccessPrivateKey = accessPrivateKey
	Ed25519Keys.RefreshPublicKey = refreshPublicKey
	Ed25519Keys.RefreshPrivateKey = refreshPrivateKey
	return nil
}
func loadKey(path string, private bool) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if private {
		block, _ := pem.Decode(b)
		if block == nil || block.Type != "PRIVATE KEY" {
			return nil, errors.New("invalid PEM")
		}

		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	} else {
		block, _ := pem.Decode(b)
		if block == nil || block.Type != "PUBLIC KEY" {
			return nil, errors.New("invalid PEM")
		}

		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	}
}

var Ed25519Keys *EdDSA = &EdDSA{}
