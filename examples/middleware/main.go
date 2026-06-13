package main

import (
	"fmt"
	"log"
	"net/http"

	"gitea.app/travelog/travelog-license-verify"
)

func main() {
	// In production, load the public key from a PEM file or embed it
	pubKey, err := verify.LoadPublicKey("testdata/public.pem")
	if err != nil {
		log.Fatalf("Failed to load public key: %v", err)
	}

	// Create a Chi router
	r := http.NewServeMux()

	// Apply the license verification middleware
	// The middleware will:
	// 1. Read and verify the .lic file on every request
	// 2. Reject requests if the license is expired or revoked
	// 3. Put the verified License in the request context
	handler := verify.Middleware(
		verify.WithLicensePath("testdata/demo.lic"),
		verify.WithPublicKey(pubKey),
	)(r)

	// Protected endpoint — retrieves license from context
	r.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		lic := verify.FromContext(r.Context())
		if lic == nil {
			http.Error(w, "No license in context", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "licensed_to": "%s",
  "product":     "%s",
  "valid":       %v,
  "features": {
    "export_pdf": %v,
    "export_csv": %v
  }
}`,
			lic.CustomerName,
			lic.Product,
			lic.IsValid(),
			lic.IsFeatureEnabled("export_pdf"),
			lic.IsFeatureEnabled("export_csv"),
		)
	})

	// Optionally, you can add middleware options:
	// - WithCacheDuration(5 * time.Minute) to cache the license
	// - WithErrorHandler(customFn) for custom error responses
	// - WithInvalidHandler(customFn) for custom expired/revoked handling

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
