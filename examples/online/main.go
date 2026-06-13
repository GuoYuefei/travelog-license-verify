package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitea.app/travelog/travelog-license-verify"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <server-url> <license-key>")
		fmt.Println("Example: go run main.go http://localhost:9443 my-license-key")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	licenseKey := os.Args[2]

	client := verify.NewClient(serverURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Verify license status online
	fmt.Println("Checking license status...")
	result, err := client.Verify(ctx, licenseKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification request failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Valid:         %v\n", result.Valid)
	fmt.Printf("Expired:       %v\n", result.Expired)
	fmt.Printf("Revoked:       %v\n", result.Revoked)
	fmt.Printf("Product:       %s\n", result.Product)
	fmt.Printf("License Type:  %s\n", result.LicenseType)
	fmt.Printf("Customer:      %s\n", result.CustomerName)
	fmt.Printf("Active Devices: %d / %d\n", result.ActiveDevices, result.MaxDevices)

	// 2. Simulate device activation (in production, this would use a real device fingerprint)
	if result.Valid {
		fmt.Println("\nActivating device...")
		actResult, err := client.Activate(ctx, verify.ActivateRequest{
			LicenseKey:        licenseKey,
			DeviceFingerprint: "cpu-mobo-mac-hash-example",
			Hostname:          "my-workstation",
			Platform:          "windows",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Activation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Activation status: %s\n", actResult.Status)
	}

	// 3. Send heartbeat
	fmt.Println("\nSending heartbeat...")
	hbResult, err := client.Heartbeat(ctx, licenseKey, "cpu-mobo-mac-hash-example")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Heartbeat failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Heartbeat status: %s\n", hbResult.Status)
}
