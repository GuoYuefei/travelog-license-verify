package verify

import (
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/pem"
	"errors"
	"os"
)

var (
	ErrInvalidPublicKey  = errors.New("verify: invalid public key")
	ErrInvalidSignature  = errors.New("verify: invalid signature")
	ErrKeyMismatch       = errors.New("verify: public key in .lic does not match provided key")
)

// PublicKeySize is the size of an Ed25519 public key in bytes.
const PublicKeySize = ed25519.PublicKeySize

// ParsePublicKey parses an Ed25519 public key from PEM-encoded data.
// Supports both "ED25519 PUBLIC KEY" and "PUBLIC KEY" PEM types.
func ParsePublicKey(pemData []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, ErrInvalidPublicKey
	}
	if block.Type != "ED25519 PUBLIC KEY" && block.Type != "PUBLIC KEY" {
		return nil, ErrInvalidPublicKey
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return ed25519.PublicKey(block.Bytes), nil
}

// LoadPublicKey reads a PEM file from disk and parses the Ed25519 public key.
func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePublicKey(data)
}

// ParsePublicKeyRaw parses a raw (non-PEM) Ed25519 public key from a byte slice.
// The key must be exactly 32 bytes.
func ParsePublicKeyRaw(raw []byte) (ed25519.PublicKey, error) {
	if len(raw) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return ed25519.PublicKey(raw), nil
}

// Verify reports whether sig is a valid Ed25519 signature of data by pub.
func Verify(pub ed25519.PublicKey, data, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	if !ed25519.Verify(pub, data, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// VerifyKeyMatch checks that the provided public key matches the expected key.
// Uses constant-time comparison.
func VerifyKeyMatch(given, expected ed25519.PublicKey) error {
	if subtle.ConstantTimeCompare(given, expected) != 1 {
		return ErrKeyMismatch
	}
	return nil
}

// GenerateKey generates a new Ed25519 key pair.
// This is provided for testing and key generation scenarios.
func GenerateKey() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	return priv, pub, err
}

// PublicKeyToPEM encodes an Ed25519 public key to PEM format.
func PublicKeyToPEM(pub ed25519.PublicKey) []byte {
	block := &pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pub,
	}
	return pem.EncodeToMemory(block)
}
