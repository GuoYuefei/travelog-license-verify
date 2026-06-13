package verify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	header    = "-----BEGIN TRAVELOG LICENSE-----"
	footer    = "-----END TRAVELOG LICENSE-----"
	formatVer = "travelog-license-v1"
	algorithm = "Ed25519"
)

var (
	// ErrInvalidFormat is returned when the .lic file has an invalid format.
	ErrInvalidFormat = errors.New("verify: invalid license file format")

	// ErrMissingField is returned when a required field is missing from the .lic file.
	ErrMissingField = errors.New("verify: missing required field in license file")
)

// LicenseFile represents a parsed .lic file before signature verification.
// It contains the raw envelope fields from the PEM-style format.
type LicenseFile struct {
	// Format is the license file format version (e.g. "travelog-license-v1").
	Format string

	// Algorithm is the signing algorithm (e.g. "Ed25519").
	Algorithm string

	// Payload is the raw JSON payload bytes. This is what the signature covers.
	Payload []byte

	// Signature is the raw Ed25519 signature bytes.
	Signature []byte

	// SignerKey is the raw Ed25519 public key bytes embedded in the .lic file.
	SignerKey []byte
}

// ParsePayload decodes the JSON payload into a new License struct.
// This does NOT verify the signature — call Verify() or use VerifyBytes() instead.
func (lf *LicenseFile) ParsePayload() (*License, error) {
	var lic License
	if err := json.Unmarshal(lf.Payload, &lic); err != nil {
		return nil, fmt.Errorf("verify: failed to parse license payload: %w", err)
	}
	return &lic, nil
}

// Verify checks the Ed25519 signature on the payload using the provided public key.
// If pub is nil, it uses the embedded SignerKey from the .lic file.
// To require a specific trusted public key, provide it explicitly.
func (lf *LicenseFile) Verify(pub ed25519.PublicKey) error {
	key := pub
	if key == nil {
		var err error
		key, err = ParsePublicKeyRaw(lf.SignerKey)
		if err != nil {
			return fmt.Errorf("verify: invalid embedded signer key: %w", err)
		}
	}
	return Verify(key, lf.Payload, lf.Signature)
}

// Decode parses a .lic file from raw bytes into a LicenseFile struct.
// This only parses the envelope — it does NOT verify the signature.
// Use VerifyBytes() for a combined parse+verify operation.
func Decode(data []byte) (*LicenseFile, error) {
	content := strings.TrimSpace(string(data))

	if !strings.HasPrefix(content, header) {
		return nil, fmt.Errorf("%w: missing header %s", ErrInvalidFormat, header)
	}
	if !strings.HasSuffix(content, footer) {
		return nil, fmt.Errorf("%w: missing footer %s", ErrInvalidFormat, footer)
	}

	body := content[len(header):]
	body = body[:len(body)-len(footer)]
	body = strings.TrimSpace(body)

	fields := make(map[string]string)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		fields[parts[0]] = parts[1]
	}

	required := []string{"format", "algorithm", "signer_key", "payload", "signature"}
	for _, key := range required {
		if _, ok := fields[key]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingField, key)
		}
	}

	signerKey, err := base64.StdEncoding.DecodeString(fields["signer_key"])
	if err != nil {
		return nil, fmt.Errorf("verify: invalid signer_key encoding: %w", err)
	}

	payload, err := base64.StdEncoding.DecodeString(fields["payload"])
	if err != nil {
		return nil, fmt.Errorf("verify: invalid payload encoding: %w", err)
	}

	signature, err := base64.StdEncoding.DecodeString(fields["signature"])
	if err != nil {
		return nil, fmt.Errorf("verify: invalid signature encoding: %w", err)
	}

	return &LicenseFile{
		Format:    fields["format"],
		Algorithm: fields["algorithm"],
		Payload:   payload,
		Signature: signature,
		SignerKey: signerKey,
	}, nil
}

// VerifyBytes parses and verifies a .lic file in one step.
// It decodes the .lic format, verifies the Ed25519 signature, and parses
// the JSON payload into a License struct.
//
// If pubKey is nil, the embedded public key from the .lic file is used
// for verification. To enforce a specific trusted public key, provide it.
//
// Returns the verified License, or an error if parsing or verification fails.
func VerifyBytes(licData []byte, pubKey ed25519.PublicKey) (*License, error) {
	lf, err := Decode(licData)
	if err != nil {
		return nil, err
	}

	if err := lf.Verify(pubKey); err != nil {
		return nil, err
	}

	lic, err := lf.ParsePayload()
	if err != nil {
		return nil, err
	}

	return lic, nil
}

// VerifyFile reads a .lic file from disk, parses it, and verifies the
// Ed25519 signature. Returns the verified License.
//
// If pubKey is nil, the embedded public key from the .lic file is used.
func VerifyFile(licPath string, pubKey ed25519.PublicKey) (*License, error) {
	data, err := os.ReadFile(licPath)
	if err != nil {
		return nil, fmt.Errorf("verify: cannot read license file %s: %w", licPath, err)
	}
	return VerifyBytes(data, pubKey)
}

// VerifyFileWithKeyFile is the simplest verification entry point.
// It reads both the .lic file and PEM public key file from disk,
// then verifies the license.
func VerifyFileWithKeyFile(licPath, pubKeyPath string) (*License, error) {
	pubKey, err := LoadPublicKey(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("verify: cannot load public key %s: %w", pubKeyPath, err)
	}
	return VerifyFile(licPath, pubKey)
}
