package verify

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupMiddlewareTest(t *testing.T, licData []byte, pubKey ed25519.PublicKey, opts ...MiddlewareOption) (http.Handler, func() *License) {
	t.Helper()

	dir := t.TempDir()
	licPath := filepath.Join(dir, "test.lic")
	err := os.WriteFile(licPath, licData, 0644)
	require.NoError(t, err)

	baseOpts := []MiddlewareOption{
		WithLicensePath(licPath),
	}
	if pubKey != nil {
		baseOpts = append(baseOpts, WithPublicKey(pubKey))
	}
	baseOpts = append(baseOpts, opts...)

	mw := Middleware(baseOpts...)

	var capturedLic *License
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLic = FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	getLic := func() *License { return capturedLic }
	return handler, getLic
}

func TestMiddleware_ValidLicense(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	handler, getLic := setupMiddlewareTest(t, licData, pub)
	require.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	captured := getLic()
	require.NotNil(t, captured)
	require.Equal(t, lic.ID, captured.ID)
	require.True(t, captured.IsFeatureEnabled("export_pdf"))
}

func TestMiddleware_ExpiredLicense(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeExpiredLicense()
	licData := generateTestLic(priv, pub, lic)

	handler, getLic := setupMiddlewareTest(t, licData, pub)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "expired")

	captured := getLic()
	require.Nil(t, captured)
}

func TestMiddleware_RevokedLicense(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	lic.RevokedAt = lic.IssuedAt + 1 // revoked
	licData := generateTestLic(priv, pub, lic)

	handler, getLic := setupMiddlewareTest(t, licData, pub)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "revoked")

	captured := getLic()
	require.Nil(t, captured)
}

func TestMiddleware_InvalidSignature(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	// tamper to make sig invalid
	lf, _ := Decode(licData)
	validSig := lf.Signature
	validSig[0] ^= 0xFF
	tamperedData := buildLicFile(pub, lf.Payload, validSig)

	handler, _ := setupMiddlewareTest(t, tamperedData, pub)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_CustomErrorHandler(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)

	var capturedErr error
	handler, _ := setupMiddlewareTest(t, licData, pub,
		WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			capturedErr = err
			http.Error(w, "custom error", http.StatusTeapot)
		}),
	)

	// Use wrong key so verification fails
	otherHandler, _ := setupMiddlewareTest(t, licData, nil,
		WithLicensePath(""), // will fail
		WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			capturedErr = err
			http.Error(w, "custom error", http.StatusTeapot)
		}),
	)
	_ = otherHandler

	// Test with a non-existent file to trigger error
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw := Middleware(
		WithLicensePath("/nonexistent/license.lic"),
		WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			capturedErr = err
			http.Error(w, "custom error", http.StatusTeapot)
		}),
	)

	handler = mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Error(t, capturedErr)
}

func TestMiddleware_CustomInvalidHandler(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeExpiredLicense()
	licData := generateTestLic(priv, pub, lic)

	var capturedLic *License
	handler, _ := setupMiddlewareTest(t, licData, pub,
		WithInvalidHandler(func(w http.ResponseWriter, r *http.Request, l *License) {
			capturedLic = l
			http.Error(w, "license not valid", http.StatusPaymentRequired)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusPaymentRequired, rec.Code)
	require.NotNil(t, capturedLic)
	require.True(t, capturedLic.IsExpired())
}

func TestFromContext_NoLicense(t *testing.T) {
	lic := FromContext(nil)
	require.Nil(t, lic)

	lic = FromContext(context.Background())
	require.Nil(t, lic)
}

func TestMiddleware_WithCache(t *testing.T) {
	pub, priv := generateTestKey()
	lic := makeValidLicense()
	licData := generateTestLic(priv, pub, lic)
	dir := t.TempDir()
	licPath := filepath.Join(dir, "test.lic")
	err := os.WriteFile(licPath, licData, 0644)
	require.NoError(t, err)

	mw := Middleware(
		WithLicensePath(licPath),
		WithPublicKey(pub),
		WithCacheDuration(0), // no cache
	)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lic := FromContext(r.Context())
		require.NotNil(t, lic)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
