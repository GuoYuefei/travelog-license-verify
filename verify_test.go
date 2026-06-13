package verify

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecode_ValidLic(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	lf, err := Decode(licData)
	require.NoError(t, err)
	require.NotNil(t, lf)
	require.Equal(t, "travelog-license-v1", lf.Format)
	require.Equal(t, "Ed25519", lf.Algorithm)
	require.Len(t, lf.SignerKey, 32)
	require.Len(t, lf.Signature, 64)
	require.NotEmpty(t, lf.Payload)
}

func TestDecode_InvalidHeader(t *testing.T) {
	_, err := Decode([]byte("garbage data"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidFormat)
}

func TestDecode_MissingField(t *testing.T) {
	data := `-----BEGIN TRAVELOG LICENSE-----
format: travelog-license-v1
-----END TRAVELOG LICENSE-----`
	_, err := Decode([]byte(data))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMissingField)
}

func TestDecode_InvalidBase64(t *testing.T) {
	data := `-----BEGIN TRAVELOG LICENSE-----
format: travelog-license-v1
algorithm: Ed25519
signer_key: !@#$%
payload: aGVsbG8=
signature: aGVsbG8=
-----END TRAVELOG LICENSE-----`
	_, err := Decode([]byte(data))
	require.Error(t, err)
	require.Contains(t, err.Error(), "signer_key")
}

func TestParsePayload(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	lf, err := Decode(licData)
	require.NoError(t, err)

	parsed, err := lf.ParsePayload()
	require.NoError(t, err)
	require.Equal(t, lic.ID, parsed.ID)
	require.Equal(t, lic.Product, parsed.Product)
	require.Equal(t, lic.CustomerName, parsed.CustomerName)
	require.Equal(t, lic.MaxDevices, parsed.MaxDevices)
	require.True(t, parsed.Features["export_pdf"])
	require.False(t, parsed.Features["bulk_import"])
}

func TestVerifyBytes_Valid(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	result, err := VerifyBytes(licData, pub)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, lic.ID, result.ID)
}

func TestVerifyBytes_WithNilKey(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	// nil key → should use embedded key from .lic file
	result, err := VerifyBytes(licData, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, lic.ID, result.ID)
}

func TestVerifyBytes_WrongKey(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	wrongPub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, err = VerifyBytes(licData, wrongPub)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerifyBytes_TamperedPayload(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	// Tamper with the payload base64
	tampered := string(licData)
	tampered = tampered[:len(tampered)-len("-----END TRAVELOG LICENSE-----\n")-5] + "XXXXX" + "\n-----END TRAVELOG LICENSE-----\n"

	_, err := VerifyBytes([]byte(tampered), pub)
	require.Error(t, err)
	// base64 decode will likely fail
}

func TestVerifyBytes_InvalidSignature(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	// modify signature to be invalid
	lf, _ := Decode(licData)
	validSig := lf.Signature
	validSig[0] ^= 0xFF // flip bits

	// rebuild .lic with broken signature
	data := buildLicFile(pub, lf.Payload, validSig)
	_, err := VerifyBytes(data, pub)
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerifyFile(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	dir := t.TempDir()
	licPath := filepath.Join(dir, "test.lic")
	err := os.WriteFile(licPath, licData, 0644)
	require.NoError(t, err)

	result, err := VerifyFile(licPath, pub)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, lic.ID, result.ID)
}

func TestVerifyFile_NotFound(t *testing.T) {
	_, err := VerifyFile("/nonexistent/license.lic", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot read license file")
}

func TestVerifyFileWithKeyFile(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	dir := t.TempDir()
	licPath := filepath.Join(dir, "test.lic")
	keyPath := filepath.Join(dir, "public.pem")

	err := os.WriteFile(licPath, licData, 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, PublicKeyToPEM(pub), 0644)
	require.NoError(t, err)

	result, err := VerifyFileWithKeyFile(licPath, keyPath)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestLicenseFile_VerifyWithNilKey(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	lf, err := Decode(licData)
	require.NoError(t, err)

	// Verify with nil key - should use embedded key
	err = lf.Verify(nil)
	require.NoError(t, err)
}

func TestLicenseFile_VerifyWithExplicitKey(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	lf, err := Decode(licData)
	require.NoError(t, err)

	// Verify with explicit key
	err = lf.Verify(pub)
	require.NoError(t, err)
}
