package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/diploma/auth-service/internal/domain"
	pkgjwt "github.com/diploma/pkg/jwt"
	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type AuthUsecase interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.TokenPair, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.TokenPair, error)
	Refresh(ctx context.Context, req *domain.RefreshRequest) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	ValidateToken(tokenStr string) (*pkgjwt.Claims, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type Handler struct {
	usecase AuthUsecase
	logger  *zap.Logger

	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func NewHandler(usecase AuthUsecase, logger *zap.Logger) *Handler {
	requestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_http_requests_total",
		Help: "Total number of HTTP requests to auth service",
	}, []string{"method", "path", "status"})

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auth_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	prometheus.MustRegister(requestsTotal, requestDuration)

	return &Handler{
		usecase:         usecase,
		logger:          logger,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
	}
}

func (h *Handler) Register(router *gin.Engine) {
	router.GET("/health", h.health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/auth")
	{
		v1.POST("/register", h.register)
		v1.POST("/login", h.login)
		v1.POST("/refresh", h.refresh)
		v1.POST("/validate", h.validate)
		v1.POST("/logout", h.authMiddleware(), h.logout)
		v1.POST("/logout-all", h.authMiddleware(), h.logoutAll)
		v1.GET("/me", h.authMiddleware(), h.me)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "auth-service",
		"timestamp": time.Now().UTC(),
	})
}

func (h *Handler) register(c *gin.Context) {
	start := time.Now()
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.requestsTotal.WithLabelValues("POST", "/register", "400").Inc()
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.usecase.Register(c.Request.Context(), &req)
	if err != nil {
		status := "500"
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			status = "409"
			h.requestsTotal.WithLabelValues("POST", "/register", status).Inc()
			response.Conflict(c, "user with this email or username already exists")
			return
		}
		h.requestsTotal.WithLabelValues("POST", "/register", status).Inc()
		h.logger.Error("register failed", zap.Error(err))
		response.InternalError(c, "registration failed")
		return
	}

	h.requestsTotal.WithLabelValues("POST", "/register", "201").Inc()
	h.requestDuration.WithLabelValues("POST", "/register").Observe(time.Since(start).Seconds())
	response.Created(c, tokens)
}

func (h *Handler) login(c *gin.Context) {
	start := time.Now()
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.usecase.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Unauthorized(c, "invalid email or password")
			return
		}
		if errors.Is(err, domain.ErrUserInactive) {
			response.Forbidden(c, "account is inactive")
			return
		}
		h.logger.Error("login failed", zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	h.requestDuration.WithLabelValues("POST", "/login").Observe(time.Since(start).Seconds())
	response.OK(c, tokens)
}

func (h *Handler) refresh(c *gin.Context) {
	var req domain.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.usecase.Refresh(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) || errors.Is(err, domain.ErrTokenExpired) {
			response.Unauthorized(c, "invalid or expired refresh token")
			return
		}
		h.logger.Error("refresh failed", zap.Error(err))
		response.InternalError(c, "token refresh failed")
		return
	}

	response.OK(c, tokens)
}

func (h *Handler) validate(c *gin.Context) {
	type validateRequest struct {
		Token string `json:"token" binding:"required"`
	}

	var req validateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	claims, err := h.usecase.ValidateToken(req.Token)
	if err != nil {
		response.Unauthorized(c, "invalid token")
		return
	}

	response.OK(c, gin.H{
		"valid":   true,
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}

func (h *Handler) logout(c *gin.Context) {
	var req domain.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.usecase.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout failed", zap.Error(err))
		response.InternalError(c, "logout failed")
		return
	}

	response.OKMessage(c, "logged out successfully")
}

func (h *Handler) logoutAll(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.usecase.LogoutAll(c.Request.Context(), uid); err != nil {
		response.InternalError(c, "logout failed")
		return
	}

	response.OKMessage(c, "logged out from all devices")
}

func (h *Handler) me(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.usecase.GetProfile(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalError(c, "failed to get profile")
		return
	}

	response.OK(c, user.ToDTO())
}

func (h *Handler) authMiddleware() gin.HandlerFunc {
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

		claims, err := h.usecase.ValidateToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}
