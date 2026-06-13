package main

import (
	"fmt"
	"os"

	"home.naturgift.fun/aiwork/travelog-license-verify"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <license.lic> <public-key.pem>")
		os.Exit(1)
	}

	licPath := os.Args[1]
	keyPath := os.Args[2]

	// Simplest one-liner: loads both files and verifies
	lic, err := verify.VerifyFileWithKeyFile(licPath, keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "License verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== License Verification Result ===")
	fmt.Printf("ID:           %s\n", lic.ID)
	fmt.Printf("Product:      %s\n", lic.Product)
	fmt.Printf("Type:         %s\n", lic.LicenseType)
	fmt.Printf("Customer:     %s (%s)\n", lic.CustomerName, lic.CustomerID)
	fmt.Printf("Status:       ")
	if lic.IsValid() {
		fmt.Println("VALID")
	} else if lic.IsExpired() {
		fmt.Println("EXPIRED")
	} else if lic.IsRevoked() {
		fmt.Println("REVOKED")
	}

	// Check features
	fmt.Println("\nFeatures:")
	for name, enabled := range lic.Features {
		status := "DISABLED"
		if enabled {
			status = "ENABLED"
		}
		fmt.Printf("  %-20s %s\n", name+":", status)
	}

	// Read capabilities — iterate all with type-aware display
	fmt.Println("\nCapabilities:")
	if len(lic.Capabilities) > 0 {
		for key, val := range lic.Capabilities {
			switch v := val.(type) {
			case string:
				fmt.Printf("  %-20s %s\n", key+":", v)
			case float64:
				fmt.Printf("  %-20s %v\n", key+":", v)
			case bool:
				fmt.Printf("  %-20s %v\n", key+":", v)
			default:
				fmt.Printf("  %-20s %v\n", key+":", v)
			}
		}
	} else {
		fmt.Println("  (none)")
	}

	// Check individual features
	fmt.Println("\nFeature Checks:")
	showFeature(lic, "export_pdf", "Can export PDF?")
	showFeature(lic, "bulk_import", "Can bulk import?")
}

func showFeature(lic *verify.License, name, label string) {
	fmt.Printf("  %-20s %v\n", label, lic.IsFeatureEnabled(name))
}
