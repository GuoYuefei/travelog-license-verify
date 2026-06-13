package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// test helpers — only compiled in tests

func generateTestKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("generate test key: %v", err))
	}
	return pub, priv
}

func makeTestLicense(expiresAt int64) *License {
	return &License{
		ID:           "test-lic-001",
		Product:      "travelog",
		LicenseType:  LicenseTypeStandard,
		CustomerID:   "cust-001",
		CustomerName: "测试用户",
		IssuedAt:     time.Now().Add(-24 * time.Hour).Unix(),
		ExpiresAt:    expiresAt,
		MaxDevices:   3,
		Features: map[string]bool{
			"export_pdf":  true,
			"export_csv":  true,
			"bulk_import": false,
		},
		Capabilities: map[string]any{
			"storage_gb":  float64(100),
			"tier":        "professional",
			"audit_log":   true,
		},
		Metadata: map[string]string{
			"region": "cn-beijing",
		},
	}
}

func makeExpiredLicense() *License {
	return makeTestLicense(time.Now().Add(-24 * time.Hour).Unix()) // expired yesterday
}

func makeValidLicense() *License {
	return makeTestLicense(time.Now().Add(365 * 24 * time.Hour).Unix()) // expires in 1 year
}

func makeNeverExpireLicense() *License {
	return makeTestLicense(0) // never expires
}

func signLicensePayload(priv ed25519.PrivateKey, lic *License) (payload []byte, sig []byte, err error) {
	payload, err = json.Marshal(lic)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal payload: %w", err)
	}
	sig = ed25519.Sign(priv, payload)
	return payload, sig, nil
}

func buildLicFile(pub ed25519.PublicKey, payload, sig []byte) []byte {
	var buf strings.Builder
	buf.WriteString("-----BEGIN TRAVELOG LICENSE-----\n")
	buf.WriteString(fmt.Sprintf("format: %s\n", "travelog-license-v1"))
	buf.WriteString(fmt.Sprintf("algorithm: %s\n", "Ed25519"))
	buf.WriteString(fmt.Sprintf("signer_key: %s\n", base64.StdEncoding.EncodeToString(pub)))
	buf.WriteString(fmt.Sprintf("payload: %s\n", base64.StdEncoding.EncodeToString(payload)))
	buf.WriteString(fmt.Sprintf("signature: %s\n", base64.StdEncoding.EncodeToString(sig)))
	buf.WriteString("-----END TRAVELOG LICENSE-----\n")
	return []byte(buf.String())
}

func generateTestLic(priv ed25519.PrivateKey, pub ed25519.PublicKey, lic *License) []byte {
	payload, sig, err := signLicensePayload(priv, lic)
	if err != nil {
		panic(err)
	}
	return buildLicFile(pub, payload, sig)
}
