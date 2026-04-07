package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/diploma/user-service/internal/domain"
	"github.com/diploma/user-service/internal/repository/postgres"
	"github.com/diploma/pkg/logger"
	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8082")
	viper.SetDefault("LOG_LEVEL", "info")

	log, _ := logger.New(viper.GetString("LOG_LEVEL"))
	defer log.Sync()

	pool, err := pgxpool.New(context.Background(), viper.GetString("DATABASE_URL"))
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("postgres ping failed", zap.Error(err))
	}

	if err := runMigrations(context.Background(), pool); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}

	repo := postgres.NewUserRepository(pool)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user-service"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/users")
	{
		v1.GET("", func(c *gin.Context) {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			filter := domain.UserFilter{
				Search: c.Query("search"),
				Role:   c.Query("role"),
				Page:   page,
				Limit:  limit,
			}
			users, total, err := repo.List(c.Request.Context(), filter)
			if err != nil {
				response.InternalError(c, "failed to get users")
				return
			}
			if users == nil {
				users = []*domain.User{}
			}
			response.Paginated(c, users, total, page, limit)
		})

		v1.GET("/:id", func(c *gin.Context) {
			id, err := uuid.Parse(c.Param("id"))
			if err != nil {
				response.BadRequest(c, "invalid user id")
				return
			}
			user, err := repo.GetByID(c.Request.Context(), id)
			if err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					response.NotFound(c, "user not found")
					return
				}
				response.InternalError(c, "failed to get user")
				return
			}
			response.OK(c, user)
		})

		v1.PUT("/:id", func(c *gin.Context) {
			requestingUserID := c.GetHeader("X-User-ID")
			requestingRole := c.GetHeader("X-User-Role")
			targetID := c.Param("id")

			if requestingRole != "admin" && requestingUserID != targetID {
				response.Forbidden(c, "cannot update other users")
				return
			}

			id, err := uuid.Parse(targetID)
			if err != nil {
				response.BadRequest(c, "invalid user id")
				return
			}

			var req domain.UpdateUserRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				response.BadRequest(c, err.Error())
				return
			}

			user, err := repo.Update(c.Request.Context(), id, &req)
			if err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					response.NotFound(c, "user not found")
					return
				}
				response.InternalError(c, "failed to update user")
				return
			}
			response.OK(c, user)
		})

		v1.DELETE("/:id", func(c *gin.Context) {
			id, err := uuid.Parse(c.Param("id"))
			if err != nil {
				response.BadRequest(c, "invalid user id")
				return
			}
			if err := repo.SetActive(c.Request.Context(), id, false); err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					response.NotFound(c, "user not found")
					return
				}
				response.InternalError(c, "failed to deactivate user")
				return
			}
			response.OKMessage(c, "user deactivated successfully")
		})

		v1.POST("/sync", func(c *gin.Context) {
			var user domain.User
			if err := c.ShouldBindJSON(&user); err != nil {
				response.BadRequest(c, err.Error())
				return
			}
			if err := repo.Upsert(c.Request.Context(), &user); err != nil {
				response.InternalError(c, "failed to sync user")
				return
			}
			response.OK(c, user)
		})
	}

	srv := &http.Server{
		Addr:         ":" + viper.GetString("PORT"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("user service started", zap.String("port", viper.GetString("PORT")))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	dirs := []string{"/migrations", "migrations"}
	var dir string
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			dir = d
			break
		}
	}
	if dir == "" {
		return nil
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, f := range files {
		var count int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=$1", f).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", f, err)
		}
		if count > 0 {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", f); err != nil {
			return fmt.Errorf("record migration %s: %w", f, err)
		}
	}
	return nil
}
