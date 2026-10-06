package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/planly/pkg/httpx"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages per-IP rate limiters.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	rate    rate.Limit
	burst   int
}

// NewIPRateLimiter creates a new rate limiter with the given requests/sec and burst capacity.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    r,
		burst:   b,
	}

	// Clean up stale entries every 5 minutes
	go limiter.cleanupStale(5 * time.Minute)

	return limiter
}

func (i *IPRateLimiter) getClient(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	c, exists := i.clients[ip]
	if !exists {
		l := rate.NewLimiter(i.rate, i.burst)
		i.clients[ip] = &clientLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (i *IPRateLimiter) cleanupStale(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		i.mu.Lock()
		for ip, c := range i.clients {
			if time.Since(c.lastSeen) > 3*interval {
				delete(i.clients, ip)
			}
		}
		i.mu.Unlock()
	}
}

// Middleware returns an HTTP handler middleware enforcing the per-IP rate limit.
func (i *IPRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			limiter := i.getClient(ip)

			if !limiter.Allow() {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Rate limit exceeded. Please slow down.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
