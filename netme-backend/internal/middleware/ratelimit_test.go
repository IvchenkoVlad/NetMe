package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/middleware"
	"golang.org/x/time/rate"
)

func TestRateLimiterBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Burst of 3 — 4th request from same IP must get 429
	r.POST("/test", middleware.RateLimiter(rate.Limit(100), 3), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 1; i <= 4; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "1.2.3.4:9999"
		r.ServeHTTP(w, req)

		if i <= 3 && w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
		if i == 4 && w.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i, w.Code)
		}
	}
}

func TestRateLimiterDifferentIPsNotBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test", middleware.RateLimiter(rate.Limit(100), 1), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i, ip := range []string{"1.2.3.4:0", "5.6.7.8:0", "9.10.11.12:0"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d from %s: expected 200, got %d", i+1, ip, w.Code)
		}
	}
}
