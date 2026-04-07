package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type TokenValidationResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Valid   bool   `json:"valid"`
		UserID  string `json:"user_id"`
		Email   string `json:"email"`
		Role    string `json:"role"`
	} `json:"data"`
}

func Auth(authServiceURL string, log *zap.Logger) gin.HandlerFunc {
	client := &http.Client{Timeout: 5 * time.Second}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			return
		}

		tokenStr := parts[1]

		reqBody := fmt.Sprintf(`{"token":"%s"}`, tokenStr)
		req, err := http.NewRequestWithContext(c.Request.Context(), "POST",
			authServiceURL+"/api/v1/auth/validate",
			strings.NewReader(reqBody),
		)
		if err != nil {
			log.Error("failed to create validate request", zap.Error(err))
			response.InternalError(c, "authentication failed")
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Error("auth service unreachable", zap.Error(err))
			response.InternalError(c, "authentication service unavailable")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			response.Unauthorized(c, "invalid or expired token")
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			response.InternalError(c, "authentication failed")
			return
		}

		var validation TokenValidationResponse
		if err := json.Unmarshal(body, &validation); err != nil || !validation.Data.Valid {
			response.Unauthorized(c, "invalid token")
			return
		}

		c.Set("user_id", validation.Data.UserID)
		c.Set("email", validation.Data.Email)
		c.Set("role", validation.Data.Role)
		c.Request.Header.Set("X-User-ID", validation.Data.UserID)
		c.Request.Header.Set("X-User-Email", validation.Data.Email)
		c.Request.Header.Set("X-User-Role", validation.Data.Role)

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "admin" {
			response.Forbidden(c, "admin access required")
			return
		}
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if query != "" {
			path = path + "?" + query
		}

		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("user_id", c.GetString("user_id")),
		)
	}
}

type ipRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    sync.RWMutex
	r     rate.Limit
	burst int
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		r:     r,
		burst: burst,
	}
}

func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.burst)
		i.ips[ip] = limiter
	}

	return limiter
}

func RateLimit(rps, burst int) gin.HandlerFunc {
	limiter := newIPRateLimiter(rate.Limit(rps), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)

		if !l.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}

func RateLimitRedis(rdb *redis.Client, rps int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s", ip)

		pipe := rdb.Pipeline()
		incr := pipe.Incr(context.Background(), key)
		pipe.Expire(context.Background(), key, time.Second)
		_, err := pipe.Exec(context.Background())

		if err != nil {
			c.Next()
			return
		}

		if incr.Val() > int64(rps) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
