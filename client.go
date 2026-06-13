package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

// DefaultHTTPTimeout is the default timeout for HTTP requests to the license server.
const DefaultHTTPTimeout = 30 * time.Second

// VerifyResult is the response from the server-side license verification endpoint.
type VerifyResult struct {
	Valid         bool              `json:"valid"`
	Expired       bool              `json:"expired"`
	Revoked       bool              `json:"revoked"`
	ExpiresAt     int64             `json:"expires_at"`
	MaxDevices    int               `json:"max_devices"`
	ActiveDevices int               `json:"active_devices"`
	Product       string            `json:"product,omitempty"`
	LicenseType   string            `json:"license_type,omitempty"`
	CustomerID    string            `json:"customer_id,omitempty"`
	CustomerName  string            `json:"customer_name,omitempty"`
	IssuedAt      int64             `json:"issued_at,omitempty"`
	Features      map[string]bool   `json:"features,omitempty"`
	Capabilities  map[string]any    `json:"capabilities,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// ActivateRequest is the request body for device activation.
type ActivateRequest struct {
	LicenseKey        string `json:"license_key"`
	DeviceFingerprint string `json:"device_fingerprint"`
	Hostname          string `json:"hostname,omitempty"`
	Platform          string `json:"platform,omitempty"`
}

// ActivateResult is the response from the device activation endpoint.
type ActivateResult struct {
	Status string `json:"status"`
	Device any    `json:"device,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HeartbeatResult is the response from the heartbeat endpoint.
type HeartbeatResult struct {
	Status         string `json:"status"`
	LicenseExpired bool   `json:"license_expired,omitempty"`
	LicenseRevoked bool   `json:"license_revoked,omitempty"`
	ExpiresAt      int64  `json:"expires_at,omitempty"`
	Device         any    `json:"device,omitempty"`
	Error          string `json:"error,omitempty"`
}

// Client is an HTTP client for the Travelog License Server's public API.
// It provides methods for online license verification, device activation,
// and heartbeat.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Client that communicates with the license server
// at the given base URL (e.g. "http://localhost:9443").
//
// The client uses a default timeout of 30 seconds. Use NewClientWithHTTP
// to provide a custom HTTP client.
func NewClient(serverURL string) *Client {
	return &Client{
		baseURL:    serverURL,
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
	}
}

// NewClientWithHTTP creates a new Client with a custom HTTP client.
func NewClientWithHTTP(serverURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    serverURL,
		httpClient: httpClient,
	}
}

// Verify calls the server-side license verification endpoint.
// GET /api/v1/client/verify/{licenseKey}
func (c *Client) Verify(ctx context.Context, licenseKey string) (*VerifyResult, error) {
	u, err := url.JoinPath(c.baseURL, "/api/v1/client/verify/", url.PathEscape(licenseKey))
	if err != nil {
		return nil, fmt.Errorf("verify client: invalid URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("verify client: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify client: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("verify client: read response: %w", err)
	}

	var result VerifyResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("verify client: parse response: %w", err)
	}

	return &result, nil
}

// Activate sends a device activation request to the license server.
// POST /api/v1/client/activate
func (c *Client) Activate(ctx context.Context, req ActivateRequest) (*ActivateResult, error) {
	u, err := url.JoinPath(c.baseURL, "/api/v1/client/activate")
	if err != nil {
		return nil, fmt.Errorf("verify client: invalid URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("verify client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("verify client: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("verify client: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("verify client: read response: %w", err)
	}

	var result ActivateResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("verify client: parse response: %w", err)
	}

	return &result, nil
}

// Heartbeat sends a heartbeat request to keep a device activation alive.
// POST /api/v1/client/heartbeat
func (c *Client) Heartbeat(ctx context.Context, licenseKey, deviceFingerprint string) (*HeartbeatResult, error) {
	u, err := url.JoinPath(c.baseURL, "/api/v1/client/heartbeat")
	if err != nil {
		return nil, fmt.Errorf("verify client: invalid URL: %w", err)
	}

	reqBody := map[string]string{
		"license_key":        licenseKey,
		"device_fingerprint": deviceFingerprint,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("verify client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("verify client: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("verify client: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("verify client: read response: %w", err)
	}

	var result HeartbeatResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("verify client: parse response: %w", err)
	}

	return &result, nil
}

// Hostname returns the system hostname.
// Used internally by ActivateLocalDevice and HeartbeatLocalDevice.
func Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

// Platform returns the OS name as detected by Go runtime (runtime.GOOS).
// Returns one of: "windows", "linux", "darwin", "freebsd", etc.
func Platform() string {
	return runtime.GOOS
}

// ActivateLocalDevice detects the local device fingerprint, hostname, and
// platform automatically, then sends an activation request to the server.
//
// This is the simplest way to activate the current machine:
//
//	result, err := client.ActivateLocalDevice(ctx, licenseKey)
func (c *Client) ActivateLocalDevice(ctx context.Context, licenseKey string) (*ActivateResult, error) {
	fp, err := DeviceFingerprint(ctx)
	if err != nil {
		return nil, fmt.Errorf("verify client: cannot get device fingerprint: %w", err)
	}

	return c.Activate(ctx, ActivateRequest{
		LicenseKey:        licenseKey,
		DeviceFingerprint: fp,
		Hostname:          Hostname(),
		Platform:          Platform(),
	})
}

// HeartbeatLocalDevice detects the local device fingerprint automatically
// and sends a heartbeat request to keep the activation alive.
//
//	result, err := client.HeartbeatLocalDevice(ctx, licenseKey)
func (c *Client) HeartbeatLocalDevice(ctx context.Context, licenseKey string) (*HeartbeatResult, error) {
	fp, err := DeviceFingerprint(ctx)
	if err != nil {
		return nil, fmt.Errorf("verify client: cannot get device fingerprint: %w", err)
	}

	return c.Heartbeat(ctx, licenseKey, fp)
}
