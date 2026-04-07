package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diploma/api-gateway/internal/config"
	"github.com/diploma/api-gateway/internal/middleware"
	"github.com/diploma/api-gateway/internal/proxy"
	"github.com/diploma/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer log.Sync()

	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		log.Fatal("failed to parse redis URL", zap.Error(err))
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn("redis unavailable, using in-memory rate limiter", zap.Error(err))
		rdb = nil
	}

	authProxy, err := proxy.New(cfg.Services.AuthURL, log)
	if err != nil {
		log.Fatal("failed to create auth proxy", zap.Error(err))
	}
	userProxy, err := proxy.New(cfg.Services.UserURL, log)
	if err != nil {
		log.Fatal("failed to create user proxy", zap.Error(err))
	}
	productProxy, err := proxy.New(cfg.Services.ProductURL, log)
	if err != nil {
		log.Fatal("failed to create product proxy", zap.Error(err))
	}
	orderProxy, err := proxy.New(cfg.Services.OrderURL, log)
	if err != nil {
		log.Fatal("failed to create order proxy", zap.Error(err))
	}
	inventoryProxy, err := proxy.New(cfg.Services.InventoryURL, log)
	if err != nil {
		log.Fatal("failed to create inventory proxy", zap.Error(err))
	}
	paymentProxy, err := proxy.New(cfg.Services.PaymentURL, log)
	if err != nil {
		log.Fatal("failed to create payment proxy", zap.Error(err))
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(log))

	if rdb != nil {
		router.Use(middleware.RateLimitRedis(rdb, cfg.RateLimit.RequestsPerSecond))
	} else {
		router.Use(middleware.RateLimit(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.BurstSize))
	}

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/health", proxy.HealthCheck(map[string]string{
		"auth-service":      cfg.Services.AuthURL,
		"user-service":      cfg.Services.UserURL,
		"product-service":   cfg.Services.ProductURL,
		"order-service":     cfg.Services.OrderURL,
		"inventory-service": cfg.Services.InventoryURL,
		"payment-service":   cfg.Services.PaymentURL,
	}, log))

	authMiddleware := middleware.Auth(cfg.Services.AuthURL, log)
	adminMiddleware := middleware.AdminOnly()

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authProxy.Handler())
			auth.POST("/login", authProxy.Handler())
			auth.POST("/refresh", authProxy.Handler())
			auth.POST("/logout", authMiddleware, authProxy.Handler())
			auth.POST("/logout-all", authMiddleware, authProxy.Handler())
			auth.GET("/me", authMiddleware, authProxy.Handler())
		}

		users := v1.Group("/users", authMiddleware)
		{
			users.GET("", adminMiddleware, userProxy.Handler())
			users.GET("/:id", userProxy.Handler())
			users.PUT("/:id", userProxy.Handler())
			users.DELETE("/:id", adminMiddleware, userProxy.Handler())
		}

		products := v1.Group("/products")
		{
			products.GET("", productProxy.Handler())
			products.GET("/search", productProxy.Handler())
			products.GET("/:id", productProxy.Handler())
			products.POST("", authMiddleware, adminMiddleware, productProxy.Handler())
			products.PUT("/:id", authMiddleware, adminMiddleware, productProxy.Handler())
			products.DELETE("/:id", authMiddleware, adminMiddleware, productProxy.Handler())
			products.POST("/:id/images", authMiddleware, adminMiddleware, productProxy.Handler())
		}

		orders := v1.Group("/orders", authMiddleware)
		{
			orders.GET("", orderProxy.Handler())
			orders.POST("", orderProxy.Handler())
			orders.GET("/:id", orderProxy.Handler())
			orders.PUT("/:id/status", adminMiddleware, orderProxy.Handler())
			orders.DELETE("/:id", orderProxy.Handler())
		}

		inventory := v1.Group("/inventory", authMiddleware)
		{
			inventory.GET("", adminMiddleware, inventoryProxy.Handler())
			inventory.GET("/:productId", inventoryProxy.Handler())
			inventory.PUT("/:productId", adminMiddleware, inventoryProxy.Handler())
		}

		payments := v1.Group("/payments", authMiddleware)
		{
			payments.GET("", paymentProxy.Handler())
			payments.POST("", paymentProxy.Handler())
			payments.GET("/:id", paymentProxy.Handler())
		}
	}

	router.NoRoute(proxy.NotFound())

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("api gateway started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down api gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
	}

	log.Info("api gateway stopped")
}
