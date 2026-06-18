package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"golang.org/x/time/rate"
)

type ipLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPLimiterStore(r rate.Limit, b int) *ipLimiterStore {
	return &ipLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (s *ipLimiterStore) allow(ip string) bool {
	s.mu.Lock()
	l, ok := s.limiters[ip]
	if !ok {
		l = rate.NewLimiter(s.r, s.b)
		s.limiters[ip] = l
	}
	s.mu.Unlock()
	return l.Allow()
}

// RateLimiter returns a per-IP token-bucket rate limiting middleware.
// r is the sustained rate (events per second); b is the burst size.
func RateLimiter(r rate.Limit, b int) gin.HandlerFunc {
	store := newIPLimiterStore(r, b)
	return func(c *gin.Context) {
		if !store.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many attempts. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
