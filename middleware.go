package verify

import (
	"context"
	"crypto/ed25519"
	"log"
	"net/http"
	"sync"
	"time"
)

type contextKey string

const licenseContextKey contextKey = "travelog-license"

// MiddlewareConfig holds the configuration for the Chi middleware.
type MiddlewareConfig struct {
	// PublicKey is the Ed25519 public key used to verify the license signature.
	// If nil, the embedded key in the .lic file is used.
	PublicKey ed25519.PublicKey

	// LicensePath is the path to the .lic file on disk.
	LicensePath string

	// CacheDuration controls how long to cache the verified license in memory.
	// If 0, the license is re-read and re-verified on every request.
	// Use a positive duration to avoid disk I/O on every request.
	CacheDuration time.Duration

	// OnError is called when license verification fails.
	// If nil, a default error handler is used that returns 500 with a JSON error.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// OnInvalid is called when the license is expired or revoked.
	// If nil, a default handler returns 403 with a JSON error.
	OnInvalid func(w http.ResponseWriter, r *http.Request, lic *License)
}

// MiddlewareOption configures the middleware.
type MiddlewareOption func(*MiddlewareConfig)

// WithPublicKey sets the Ed25519 public key for signature verification.
func WithPublicKey(pub ed25519.PublicKey) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.PublicKey = pub
	}
}

// WithLicensePath sets the path to the .lic file.
func WithLicensePath(path string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.LicensePath = path
	}
}

// WithCacheDuration sets how long to cache the verified license.
func WithCacheDuration(d time.Duration) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.CacheDuration = d
	}
}

// WithErrorHandler sets a custom error handler for verification failures.
func WithErrorHandler(fn func(w http.ResponseWriter, r *http.Request, err error)) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.OnError = fn
	}
}

// WithInvalidHandler sets a custom handler for expired/revoked licenses.
func WithInvalidHandler(fn func(w http.ResponseWriter, r *http.Request, lic *License)) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.OnInvalid = fn
	}
}

type cachedLicense struct {
	license   *License
	expiresAt time.Time
}

// Middleware returns an HTTP middleware (suitable for go-chi/chi or standard
// net/http) that verifies a Travelog .lic license file on every request.
//
// The verified License is stored in the request context and can be retrieved
// with FromContext(). If the license is expired or revoked, the request is
// rejected with a 403 response.
//
// Usage with Chi:
//
//	r := chi.NewRouter()
//	r.Use(verify.Middleware(
//	    verify.WithLicensePath("license.lic"),
//	    verify.WithPublicKey(pubKey),
//	))
func Middleware(opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := &MiddlewareConfig{
		CacheDuration: 0, // no caching by default
		OnError:       defaultErrorHandler,
		OnInvalid:     defaultInvalidHandler,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var (
		cache   *cachedLicense
		cacheMu sync.RWMutex
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lic, err := getLicense(cfg, &cache, &cacheMu)
			if err != nil {
				cfg.OnError(w, r, err)
				return
			}

			if !lic.IsValid() {
				cfg.OnInvalid(w, r, lic)
				return
			}

			ctx := context.WithValue(r.Context(), licenseContextKey, lic)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getLicense(cfg *MiddlewareConfig, cache **cachedLicense, mu *sync.RWMutex) (*License, error) {
	if cfg.CacheDuration > 0 {
		mu.RLock()
		if *cache != nil && time.Now().Before((*cache).expiresAt) {
			lic := (*cache).license
			mu.RUnlock()
			return lic, nil
		}
		mu.RUnlock()
	}

	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring write lock
	if cfg.CacheDuration > 0 && *cache != nil && time.Now().Before((*cache).expiresAt) {
		return (*cache).license, nil
	}

	lic, err := VerifyFile(cfg.LicensePath, cfg.PublicKey)
	if err != nil {
		return nil, err
	}

	if cfg.CacheDuration > 0 {
		*cache = &cachedLicense{
			license:   lic,
			expiresAt: time.Now().Add(cfg.CacheDuration),
		}
	}

	return lic, nil
}

// FromContext retrieves the verified License from the request context.
// Returns nil if the middleware did not run or the request was not authenticated.
func FromContext(ctx context.Context) *License {
	if ctx == nil {
		return nil
	}
	lic, _ := ctx.Value(licenseContextKey).(*License)
	return lic
}

func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("verify middleware: license verification failed: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error":"license verification failed"}`))
}

func defaultInvalidHandler(w http.ResponseWriter, r *http.Request, lic *License) {
	w.Header().Set("Content-Type", "application/json")
	if lic.IsRevoked() {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"license has been revoked"}`))
		return
	}
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"license has expired"}`))
}
