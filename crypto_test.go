package verify

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePublicKey_PEM(t *testing.T) {
	pub, _ := generateTestKey()
	pemData := PublicKeyToPEM(pub)

	parsed, err := ParsePublicKey(pemData)
	require.NoError(t, err)
	require.Equal(t, pub, parsed)
}

func TestParsePublicKey_InvalidPEM(t *testing.T) {
	_, err := ParsePublicKey([]byte("not pem"))
	require.ErrorIs(t, err, ErrInvalidPublicKey)
}

func TestParsePublicKey_InvalidLength(t *testing.T) {
	pemData := []byte("-----BEGIN PUBLIC KEY-----\n" +
		"MA==\n" +
		"-----END PUBLIC KEY-----")
	_, err := ParsePublicKey(pemData)
	require.ErrorIs(t, err, ErrInvalidPublicKey)
}

func TestParsePublicKey_WrongType(t *testing.T) {
	pemData := []byte("-----BEGIN CERTIFICATE-----\n" +
		"MA==\n" +
		"-----END CERTIFICATE-----")
	_, err := ParsePublicKey(pemData)
	require.ErrorIs(t, err, ErrInvalidPublicKey)
}

func TestParsePublicKeyRaw(t *testing.T) {
	pub, _ := generateTestKey()

	parsed, err := ParsePublicKeyRaw(pub)
	require.NoError(t, err)
	require.Equal(t, pub, parsed)
}

func TestParsePublicKeyRaw_WrongLength(t *testing.T) {
	_, err := ParsePublicKeyRaw([]byte("too short"))
	require.ErrorIs(t, err, ErrInvalidPublicKey)
}

func TestLoadPublicKey(t *testing.T) {
	pub, _ := generateTestKey()
	pemData := PublicKeyToPEM(pub)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "public.pem")
	err := os.WriteFile(keyPath, pemData, 0644)
	require.NoError(t, err)

	loaded, err := LoadPublicKey(keyPath)
	require.NoError(t, err)
	require.Equal(t, pub, loaded)
}

func TestLoadPublicKey_NotFound(t *testing.T) {
	_, err := LoadPublicKey("/nonexistent/key.pem")
	require.Error(t, err)
}

func TestVerify_ValidSignature(t *testing.T) {
	pub, priv := generateTestKey()
	data := []byte("hello world")
	sig := ed25519.Sign(priv, data)

	err := Verify(pub, data, sig)
	require.NoError(t, err)
}

func TestVerify_InvalidSignature(t *testing.T) {
	pub, _ := generateTestKey()
	data := []byte("hello world")
	sig := make([]byte, 64) // all zeros

	err := Verify(pub, data, sig)
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerify_WrongSignatureLength(t *testing.T) {
	pub, _ := generateTestKey()
	data := []byte("hello world")

	err := Verify(pub, data, []byte("short"))
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerifyKeyMatch(t *testing.T) {
	pub1, _ := generateTestKey()
	pub2, _ := generateTestKey()

	err := VerifyKeyMatch(pub1, pub1)
	require.NoError(t, err)

	err = VerifyKeyMatch(pub1, pub2)
	require.ErrorIs(t, err, ErrKeyMismatch)
}

func TestGenerateKey(t *testing.T) {
	priv, pub, err := GenerateKey()
	require.NoError(t, err)
	require.Len(t, pub, 32)
	require.Len(t, priv, 64)
}

func TestPublicKeyToPEM_RoundTrip(t *testing.T) {
	pub, _ := generateTestKey()
	pemData := PublicKeyToPEM(pub)

	parsed, err := ParsePublicKey(pemData)
	require.NoError(t, err)
	require.Equal(t, pub, parsed)
}
