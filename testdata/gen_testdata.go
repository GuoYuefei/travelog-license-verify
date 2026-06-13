//go:build ignore

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type License struct {
	ID           string            `json:"id"`
	Product      string            `json:"product"`
	LicenseType  string            `json:"license_type"`
	CustomerID   string            `json:"customer_id"`
	CustomerName string            `json:"customer_name"`
	IssuedAt     int64             `json:"issued_at"`
	ExpiresAt    int64             `json:"expires_at"`
	MaxDevices   int               `json:"max_devices"`
	Features     map[string]bool   `json:"features"`
	Capabilities map[string]any    `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RevokedAt    int64             `json:"revoked_at,omitempty"`
}

func PublicKeyToPEM(pub ed25519.PublicKey) []byte {
	pem := "-----BEGIN ED25519 PUBLIC KEY-----\n"
	// base64 encode with 64-char line wrapping
	encoded := base64.StdEncoding.EncodeToString(pub)
	for len(encoded) > 0 {
		line := encoded
		if len(line) > 64 {
			line = line[:64]
		}
		pem += line + "\n"
		encoded = encoded[len(line):]
	}
	pem += "-----END ED25519 PUBLIC KEY-----\n"
	return []byte(pem)
}

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	lic := License{
		ID:           "demo-lic-001",
		Product:      "travelog",
		LicenseType:  "standard",
		CustomerID:   "cust-demo-001",
		CustomerName: "Demo Customer",
		IssuedAt:     time.Now().Add(-7 * 24 * time.Hour).Unix(),
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour).Unix(),
		MaxDevices:   5,
		Features: map[string]bool{
			"export_pdf":   true,
			"export_csv":   true,
			"export_raw":   true,
			"bulk_import":  false,
			"watermark":    true,
		},
		Capabilities: map[string]any{
			"storage_gb":   float64(500),
			"tier":         "professional",
			"audit_log":    true,
			"api_rate":     float64(1000),
			"max_team":     float64(10),
		},
		Metadata: map[string]string{
			"region":  "cn-beijing",
			"contact": "admin@example.com",
		},
	}

	payload, err := json.Marshal(lic)
	if err != nil {
		panic(err)
	}

	sig := ed25519.Sign(priv, payload)

	licFile := fmt.Sprintf("-----BEGIN TRAVELOG LICENSE-----\n"+
		"format: travelog-license-v1\n"+
		"algorithm: Ed25519\n"+
		"signer_key: %s\n"+
		"payload: %s\n"+
		"signature: %s\n"+
		"-----END TRAVELOG LICENSE-----\n",
		base64.StdEncoding.EncodeToString(pub),
		base64.StdEncoding.EncodeToString(payload),
		base64.StdEncoding.EncodeToString(sig),
	)

	if err := os.WriteFile("testdata/demo.lic", []byte(licFile), 0644); err != nil {
		panic(err)
	}
	fmt.Println("Generated: testdata/demo.lic")

	if err := os.WriteFile("testdata/public.pem", PublicKeyToPEM(pub), 0644); err != nil {
		panic(err)
	}
	fmt.Println("Generated: testdata/public.pem")

	if err := os.WriteFile("testdata/private.pem", []byte("not needed for verification"), 0644); err != nil {
		panic(err)
	}

	fmt.Println("\nPublic key (base64):", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("License ID:", lic.ID)
}
