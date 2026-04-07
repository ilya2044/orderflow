package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	pkgkafka "github.com/diploma/pkg/kafka"
	"github.com/diploma/pkg/logger"
	"github.com/diploma/pkg/response"
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Payment struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	OrderID   uuid.UUID `db:"order_id"   json:"order_id"`
	UserID    string    `db:"user_id"    json:"user_id"`
	Amount    float64   `db:"amount"     json:"amount"`
	Status    string    `db:"status"     json:"status"`
	Method    string    `db:"method"     json:"method"`
	ExternalID string   `db:"external_id" json:"external_id,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePaymentRequest struct {
	OrderID string  `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount"   binding:"required,gt=0"`
	Method  string  `json:"method"   binding:"required,oneof=card bank_transfer sbp yookassa"`
}

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8086")
	viper.SetDefault("LOG_LEVEL", "info")

	log, _ := logger.New(viper.GetString("LOG_LEVEL"))
	defer log.Sync()

	pool, err := pgxpool.New(context.Background(), viper.GetString("DATABASE_URL"))
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres")

	if err := runMigrations(context.Background(), pool); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}

	brokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	kafkaProducer, err := pkgkafka.NewProducer(brokers, log)
	if err != nil {
		log.Error("kafka producer unavailable", zap.Error(err))
	} else {
		defer kafkaProducer.Close()
	}

	consumer, err := pkgkafka.NewConsumer(brokers, "payment-service",
		[]string{pkgkafka.TopicInventoryReserved},
		makePaymentHandler(pool, kafkaProducer, log), log)
	if err != nil {
		log.Error("kafka consumer unavailable", zap.Error(err))
	} else {
		go func() {
			consumer.Start(context.Background())
		}()
		defer consumer.Close()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "payment-service"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/payments")
	{
		v1.GET("", func(c *gin.Context) {
			userID := c.GetHeader("X-User-ID")
			role := c.GetHeader("X-User-Role")

			var rows interface{ Close() }
			var queryErr error
			if role == "admin" {
				rows, queryErr = pool.Query(c.Request.Context(),
					"SELECT id, order_id, user_id, amount, status, method, COALESCE(external_id,''), created_at, updated_at FROM payments ORDER BY created_at DESC")
			} else {
				rows, queryErr = pool.Query(c.Request.Context(),
					"SELECT id, order_id, user_id, amount, status, method, COALESCE(external_id,''), created_at, updated_at FROM payments WHERE user_id = $1 ORDER BY created_at DESC",
					userID)
			}
			if queryErr != nil {
				response.InternalError(c, "failed to get payments")
				return
			}
			defer rows.(interface{ Close() }).Close()

			payments := []Payment{}
			if pgRows, ok := rows.(pgx.Rows); ok {
				for pgRows.Next() {
					p := Payment{}
					pgRows.Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Status, &p.Method, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt)
					payments = append(payments, p)
				}
			}
			response.OK(c, payments)
		})

		v1.POST("", func(c *gin.Context) {
			userID := c.GetHeader("X-User-ID")
			var req CreatePaymentRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				response.BadRequest(c, err.Error())
				return
			}

			orderID, err := uuid.Parse(req.OrderID)
			if err != nil {
				response.BadRequest(c, "invalid order id")
				return
			}

			payment := Payment{
				ID:        uuid.New(),
				OrderID:   orderID,
				UserID:    userID,
				Amount:    req.Amount,
				Status:    "processing",
				Method:    req.Method,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			_, err = pool.Exec(c.Request.Context(),
				"INSERT INTO payments (id, order_id, user_id, amount, status, method, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
				payment.ID, payment.OrderID, payment.UserID, payment.Amount, payment.Status, payment.Method, payment.CreatedAt, payment.UpdatedAt,
			)
			if err != nil {
				response.InternalError(c, "failed to create payment")
				return
			}

			go simulatePayment(context.Background(), pool, kafkaProducer, &payment, log)

			response.Created(c, payment)
		})

		v1.GET("/:id", func(c *gin.Context) {
			id, err := uuid.Parse(c.Param("id"))
			if err != nil {
				response.BadRequest(c, "invalid payment id")
				return
			}

			p := Payment{}
			err = pool.QueryRow(c.Request.Context(),
				"SELECT id, order_id, user_id, amount, status, method, COALESCE(external_id,''), created_at, updated_at FROM payments WHERE id = $1",
				id).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Status, &p.Method, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt)
			if errors.Is(err, pgx.ErrNoRows) {
				response.NotFound(c, "payment not found")
				return
			}
			if err != nil {
				response.InternalError(c, "failed to get payment")
				return
			}
			response.OK(c, p)
		})
	}

	srv := &http.Server{
		Addr:    ":" + viper.GetString("PORT"),
		Handler: router,
	}

	go func() {
		log.Info("payment service started", zap.String("port", viper.GetString("PORT")))
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

func simulatePayment(ctx context.Context, pool *pgxpool.Pool, producer *pkgkafka.Producer, p *Payment, log *zap.Logger) {
	time.Sleep(2 * time.Second)

	status := "completed"
	_, err := pool.Exec(ctx,
		"UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2",
		status, p.ID,
	)
	if err != nil {
		log.Error("failed to update payment status", zap.Error(err))
		return
	}

	if producer != nil {
		event := pkgkafka.PaymentProcessedEvent{
			PaymentID: p.ID.String(),
			OrderID:   p.OrderID.String(),
			UserID:    p.UserID,
			Amount:    p.Amount,
			Status:    status,
			Method:    p.Method,
		}
		_ = producer.Publish(pkgkafka.TopicPaymentProcessed, p.OrderID.String(), event)
	}

	log.Info("payment processed", zap.String("payment_id", p.ID.String()), zap.String("status", status))
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

func makePaymentHandler(pool *pgxpool.Pool, producer *pkgkafka.Producer, log *zap.Logger) pkgkafka.MessageHandler {
	return func(msg *sarama.ConsumerMessage) error {
		if msg.Topic == pkgkafka.TopicInventoryReserved {
			var event pkgkafka.InventoryReservedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			if event.Reserved {
				log.Info("inventory reserved, awaiting payment", zap.String("order_id", event.OrderID))
			}
		}
		return nil
	}
}
