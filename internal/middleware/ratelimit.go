package middleware

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
)

type ipLimiterEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// AuthLoginRateLimiter returns middleware that rate-limits per client IP (auth/login and auth/register).
// rpm <= 0 disables limiting.
func AuthLoginRateLimiter(rpm int) gin.HandlerFunc {
	if rpm <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	var mu sync.Mutex
	ips := make(map[string]*ipLimiterEntry)
	limit := rate.Limit(float64(rpm) / 60.0)
	burst := max(5, rpm/6)
	if burst < 1 {
		burst = 1
	}

	return func(c *gin.Context) {
		ip := clientIP(c)
		now := time.Now().UTC()

		mu.Lock()
		if len(ips) > 20000 {
			cutoff := now.Add(-15 * time.Minute)
			for k, v := range ips {
				if v.lastSeen.Before(cutoff) {
					delete(ips, k)
				}
			}
		}
		ent, ok := ips[ip]
		if !ok {
			ent = &ipLimiterEntry{lim: rate.NewLimiter(limit, burst), lastSeen: now}
			ips[ip] = ent
		}
		ent.lastSeen = now
		allowed := ent.lim.Allow()
		mu.Unlock()

		if !allowed {
			response.Error(c, apierrors.ErrRateLimited)
			c.Abort()
			return
		}
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(c.Request.RemoteAddr)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
