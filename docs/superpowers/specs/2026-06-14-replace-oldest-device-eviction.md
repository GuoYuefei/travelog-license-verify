# Replace Oldest Device Eviction

**Date**: 2026-06-14
**Status**: Approved

## Summary

When activating a device that would exceed `MaxDevices`, the client may set `replace_oldest: true` to automatically evict the earliest-activated device and activate the new one in a single request.

## Architecture

```
Client (travelog-license-verify)          Server (travelog-license)
┌─────────────────────────┐              ┌──────────────────────────────┐
│ ActivateRequest {       │  POST        │ Activate handler:            │
│   replace_oldest: true  │  /client/    │   if replace_oldest &&       │
│ }                       │  activate    │     已达上限 →                │
│                         │ ──────────►  │     1. GetOldestActiveByLicense│
│ ActivateReplaceOldest() │             │     2. Deactivate(oldest)     │
│ (convenience method)    │ ◄──────────  │     3. Create(newDevice)      │
│                         │  {status:   │     4. Return replaced_device │
│                         │   "activated",                            │
└─────────────────────────┘  device:...,                              │
                             replaced_device:...}                      └──────────────────────────────┘
```

## Changes — Client (`travelog-license-verify`)

### 1. `ActivateRequest` — Add field

**File**: `client.go:37-43`

```go
type ActivateRequest struct {
    LicenseKey        string `json:"license_key"`
    DeviceFingerprint string `json:"device_fingerprint"`
    Hostname          string `json:"hostname,omitempty"`
    Platform          string `json:"platform,omitempty"`
    ReplaceOldest     bool   `json:"replace_oldest,omitempty"`  // ← NEW
}
```

`omitempty` means `false` (default) omits the field from JSON.

### 2. `ActivateResult` — Add field

**File**: `client.go:45-50`

```go
type ActivateResult struct {
    Status         string `json:"status"`
    Device         any    `json:"device,omitempty"`
    Error          string `json:"error,omitempty"`
    ReplacedDevice any    `json:"replaced_device,omitempty"`  // ← NEW
}
```

`ReplacedDevice` contains the evicted device object when the server performed eviction.

### 3. `ActivateReplaceOldest()` — Convenience method

**File**: `client.go` (after `ActivateLocalDevice`, line 237)

```go
// ActivateReplaceOldest activates the current machine, automatically evicting
// the oldest activated device if the maximum device count has been reached.
//
//	result, err := client.ActivateReplaceOldest(ctx, licenseKey)
func (c *Client) ActivateReplaceOldest(ctx context.Context, licenseKey string) (*ActivateResult, error) {
    fp, err := DeviceFingerprint(ctx)
    if err != nil {
        return nil, fmt.Errorf("verify client: cannot get device fingerprint: %w", err)
    }
    return c.Activate(ctx, ActivateRequest{
        LicenseKey:        licenseKey,
        DeviceFingerprint: fp,
        Hostname:          Hostname(),
        Platform:          Platform(),
        ReplaceOldest:     true,
    })
}
```

### 4. Tests

**File**: `client_test.go`

- `TestClient_Activate_ReplaceOldest`: mock server checks `replace_oldest: true`, returns `{status:"activated", replaced_device:{id:"dev-001"}}`
- Follow existing `httptest.NewServer` pattern from `TestClient_Activate_TooManyDevices`

## Changes — Server (`travelog-license`)

### 1. `DeviceRepo.GetOldestActiveByLicense()` — New repo method

**File**: `internal/repo/device_repo.go` (after line 193)

```go
// GetOldestActiveByLicense returns the earliest-activated active device
// for the given license, or nil if none exists.
func (r *DeviceRepo) GetOldestActiveByLicense(licenseID string) (*model.Device, error) {
    row := r.db.QueryRow(`SELECT id, license_id, device_fingerprint, hostname, platform,
        activated_at, last_seen_at, is_active FROM devices
        WHERE license_id = ? AND is_active = 1
        ORDER BY activated_at ASC LIMIT 1`, licenseID)
    d, err := scanDevice(row)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("get oldest active device: %w", err)
    }
    return d, nil
}
```

### 2. Activate handler — Modify device count check

**File**: `internal/handler/client.go:33-103`

**Request struct** — add field:
```go
var req struct {
    LicenseKey        string `json:"license_key"`
    DeviceFingerprint string `json:"device_fingerprint"`
    Hostname          string `json:"hostname"`
    Platform          string `json:"platform"`
    ReplaceOldest     bool   `json:"replace_oldest"`   // ← NEW
}
```

**Device count check** — add eviction branch (lines 75-84 become):
```go
if lic.MaxDevices > 0 {
    count, err := h.deviceRepo.CountActiveByLicense(req.LicenseKey)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check device count"})
        return
    }
    if count >= lic.MaxDevices {
        if req.ReplaceOldest {
            // Evict the oldest active device
            oldest, err := h.deviceRepo.GetOldestActiveByLicense(req.LicenseKey)
            if err != nil {
                writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to find oldest device"})
                return
            }
            if oldest == nil {
                writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no active device to replace"})
                return
            }
            if err := h.deviceRepo.Deactivate(oldest.ID); err != nil {
                writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to replace device"})
                return
            }
            replacedDevice = oldest  // track for response
        } else {
            writeJSON(w, http.StatusConflict, map[string]string{"error": "maximum number of devices already activated"})
            return
        }
    }
}
```

**Response** — include `replaced_device`:
```go
resp := map[string]interface{}{
    "status": "activated",
    "device": device,
}
if replacedDevice != nil {
    resp["replaced_device"] = replacedDevice
}
writeJSON(w, http.StatusOK, resp)
```

### 3. Tests

**File**: `internal/handler/client_test.go`

- `TestActivateReplaceOldest`: seed 3 devices (fill limit), activate 4th with `replace_oldest:true`, verify:
  - HTTP 200 OK
  - `status` = `"activated"`
  - `replaced_device` field exists and contains the first device's fingerprint
  - First device is now `is_active = 0` in DB

**File**: `internal/repo/repo_test.go`

- `TestDeviceGetOldestActive`: create 3 devices with different `activated_at`, verify oldest returned

## Scenarios

| ID | Scenario | Expected | Type |
|----|----------|----------|------|
| S1 | 4th device activate with `replace_oldest:true`, max_devices=3 | 200, device activated, oldest deactivated | Happy |
| S2 | 4th device activate without `replace_oldest` | 409 Conflict | Regression |
| S3 | `ActivateReplaceOldest()` convenience method sends correct request | Server sees `replace_oldest:true` | Happy |
| S4 | Normal activate still works unchanged | 200, device activated | Regression |
| S5 | GetOldestActiveByLicense with no active devices | Returns nil, nil | Edge |
