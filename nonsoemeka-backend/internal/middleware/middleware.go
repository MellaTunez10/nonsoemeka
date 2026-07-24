package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/auth"
)

type ctxKey string

const (
	CtxKeyRequestID ctxKey = "request_id"
	CtxKeyUser      ctxKey = "user_claims"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(CtxKeyRequestID).(string); ok {
		return id
	}
	return ""
}

func GetUserClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(CtxKeyUser).(*auth.Claims)
	return claims, ok
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID := GetRequestID(r.Context())
				slog.Error("panic recovered", "panic", rec, "request_id", reqID)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
					Error: apperrors.ErrorDetail{
						Code:      "INTERNAL_ERROR",
						Message:   "An internal server error occurred",
						RequestID: reqID,
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), CtxKeyRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusResponseWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

func LoggingAndMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(sw, r)

		duration := time.Since(start)
		reqID := GetRequestID(r.Context())
		statusStr := strconv.Itoa(sw.statusCode)
		routePattern := r.URL.Path

		httpRequestsTotal.WithLabelValues(r.Method, routePattern, statusStr).Inc()
		httpRequestDuration.WithLabelValues(r.Method, routePattern, statusStr).Observe(duration.Seconds())

		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
		)
	})
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedSet[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowedSet[origin] || len(allowedOrigins) == 0) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Request-ID")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimiterBackend is the interface for rate-limiting backends.
// The default implementation is in-memory (InMemoryRateLimiter).
//
// SCALING NOTE: The in-memory backend stores counters in process memory.
// This means:
//   - Counters reset on every restart / redeploy.
//   - If you run 2+ API replicas, each has its own counters and
//     effective limits are multiplied by the replica count.
//
// For multi-instance deployments, implement this interface with a
// Redis-backed sliding window (e.g. using MULTI/EXEC with ZRANGEBYSCORE).
type RateLimiterBackend interface {
	// Allow returns true if the request identified by key is within limits.
	Allow(key string) bool
}

// InMemoryRateLimiter is a sliding-window rate limiter backed by a plain
// Go map. Suitable for single-instance deployments only.
type InMemoryRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) RateLimiterBackend {
	rl := &InMemoryRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *InMemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true
}

func (rl *InMemoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimitMiddleware(limiter RateLimiterBackend) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
				clientIP = strings.Split(ip, ",")[0]
			}

			if !limiter.Allow(clientIP) {
				reqID := GetRequestID(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
					Error: apperrors.ErrorDetail{
						Code:      "TOO_MANY_REQUESTS",
						Message:   "Rate limit exceeded",
						RequestID: reqID,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.ParseAccessToken(tokenStr, jwtSecret)
			if err != nil {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid access token")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool)
	for _, role := range allowedRoles {
		roleSet[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok || claims == nil {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}

			if !roleSet[claims.Role] {
				writeError(w, r, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
	})
}

// Docker default bridge and overlay networks
var dockerCIDRs = []string{
	"172.16.0.0/12",
	"10.0.0.0/8",
	"192.168.0.0/16",
}

// InternalOnlyMiddleware restricts access to requests originating from
// localhost (127.0.0.1 / ::1) or Docker-internal networks.
// Use this to protect endpoints like /metrics from public access.
func InternalOnlyMiddleware(next http.Handler) http.Handler {
	// Pre-parse the CIDR blocks once at init time.
	var internalNets []*net.IPNet
	for _, cidr := range dockerCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			internalNets = append(internalNets, network)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP := r.RemoteAddr

		// X-Forwarded-For takes precedence when behind a proxy
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			remoteIP = strings.TrimSpace(strings.Split(xff, ",")[0])
		}

		// Strip port if present (e.g. "172.18.0.3:54321" → "172.18.0.3")
		host, _, err := net.SplitHostPort(remoteIP)
		if err == nil {
			remoteIP = host
		}

		ip := net.ParseIP(remoteIP)
		if ip == nil {
			writeError(w, r, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}

		// Allow loopback
		if ip.IsLoopback() {
			next.ServeHTTP(w, r)
			return
		}

		// Allow Docker-internal networks
		for _, network := range internalNets {
			if network.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}

		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "access denied")
	})
}
