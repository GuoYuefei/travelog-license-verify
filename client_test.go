package verify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:9443")
	require.NotNil(t, c)
	require.NotNil(t, c.httpClient)
}

func TestClient_Verify(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/client/verify/test-key", r.URL.Path)

		resp := VerifyResult{
			Valid:         true,
			Expired:       false,
			Revoked:       false,
			ExpiresAt:     1900000000,
			MaxDevices:    3,
			ActiveDevices: 1,
			Product:       "travelog",
			LicenseType:   "standard",
			CustomerName:  "Test User",
			Features:      map[string]bool{"export_pdf": true},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Verify(context.Background(), "test-key")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Valid)
	require.False(t, result.Expired)
	require.Equal(t, 3, result.MaxDevices)
	require.True(t, result.Features["export_pdf"])
}

func TestClient_Verify_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VerifyResult{
			Valid: false,
			Error: "license not found",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Verify(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Contains(t, result.Error, "not found")
}

func TestClient_Activate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/client/activate", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req ActivateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		require.Equal(t, "test-key", req.LicenseKey)
		require.Equal(t, "fp-123", req.DeviceFingerprint)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivateResult{
			Status: "activated",
			Device: map[string]any{"id": "dev-001", "fingerprint": "fp-123"},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Activate(context.Background(), ActivateRequest{
		LicenseKey:        "test-key",
		DeviceFingerprint: "fp-123",
		Hostname:          "my-pc",
		Platform:          "windows",
	})
	require.NoError(t, err)
	require.Equal(t, "activated", result.Status)
}

func TestClient_Activate_AlreadyActivated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivateResult{
			Status: "already_activated",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Activate(context.Background(), ActivateRequest{
		LicenseKey:        "test-key",
		DeviceFingerprint: "fp-123",
	})
	require.NoError(t, err)
	require.Equal(t, "already_activated", result.Status)
}

func TestClient_Activate_TooManyDevices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ActivateResult{
			Status: "",
			Error:  "maximum number of devices already activated",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Activate(context.Background(), ActivateRequest{
		LicenseKey:        "test-key",
		DeviceFingerprint: "fp-123",
	})
	require.NoError(t, err)
	require.Contains(t, result.Error, "maximum number")
}

func TestClient_Heartbeat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/client/heartbeat", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		require.Equal(t, "test-key", req["license_key"])
		require.Equal(t, "fp-123", req["device_fingerprint"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HeartbeatResult{
			Status: "ok",
			Device: map[string]any{"id": "dev-001", "fingerprint": "fp-123"},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Heartbeat(context.Background(), "test-key", "fp-123")
	require.NoError(t, err)
	require.Equal(t, "ok", result.Status)
}

func TestClient_Heartbeat_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(HeartbeatResult{
			Status: "",
			Error:  "device not found",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Heartbeat(context.Background(), "nonexistent", "fp-999")
	require.NoError(t, err)
	require.Contains(t, result.Error, "not found")
}

func TestNewClientWithHTTP(t *testing.T) {
	customClient := &http.Client{}
	c := NewClientWithHTTP("http://localhost:9443", customClient)
	require.NotNil(t, c)
	require.Equal(t, customClient, c.httpClient)
}

func TestClient_Verify_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.Verify(context.Background(), "test-key")
	require.NoError(t, err) // we parse the body regardless of status code
	require.Contains(t, result.Error, "internal error")
}
