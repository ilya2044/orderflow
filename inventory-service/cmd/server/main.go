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

type InventoryItem struct {
	ProductID   string    `db:"product_id"  json:"product_id"`
	ProductName string    `db:"product_name" json:"product_name"`
	Stock       int       `db:"stock"        json:"stock"`
	Reserved    int       `db:"reserved"     json:"reserved"`
	Available   int       `db:"available"    json:"available"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}

type UpdateStockRequest struct {
	Stock    *int   `json:"stock"`
	Delta    *int   `json:"delta"`
	Reason   string `json:"reason"`
}

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8085")
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

	topics := []string{pkgkafka.TopicOrderCreated, pkgkafka.TopicOrderCancelled, pkgkafka.TopicPaymentProcessed}
	consumer, err := pkgkafka.NewConsumer(brokers, "inventory-service", topics, makeMessageHandler(pool, kafkaProducer, log), log)
	if err != nil {
		log.Error("kafka consumer unavailable", zap.Error(err))
	} else {
		go func() {
			if err := consumer.Start(context.Background()); err != nil {
				log.Error("consumer stopped", zap.Error(err))
			}
		}()
		defer consumer.Close()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "inventory-service"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/inventory")
	{
		v1.GET("", func(c *gin.Context) {
			rows, err := pool.Query(c.Request.Context(),
				"SELECT product_id, product_name, stock, reserved, (stock - reserved) as available, updated_at FROM inventory ORDER BY product_name")
			if err != nil {
				response.InternalError(c, "failed to get inventory")
				return
			}
			defer rows.Close()

			var items []InventoryItem
			for rows.Next() {
				item := InventoryItem{}
				if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Stock, &item.Reserved, &item.Available, &item.UpdatedAt); err != nil {
					continue
				}
				items = append(items, item)
			}
			if items == nil {
				items = []InventoryItem{}
			}
			response.OK(c, items)
		})

		v1.GET("/:productId", func(c *gin.Context) {
			productID := c.Param("productId")
			item := InventoryItem{}
			err := pool.QueryRow(c.Request.Context(),
				"SELECT product_id, product_name, stock, reserved, (stock - reserved) as available, updated_at FROM inventory WHERE product_id = $1",
				productID).Scan(&item.ProductID, &item.ProductName, &item.Stock, &item.Reserved, &item.Available, &item.UpdatedAt)
			if errors.Is(err, pgx.ErrNoRows) {
				response.NotFound(c, "product not in inventory")
				return
			}
			if err != nil {
				response.InternalError(c, "failed to get inventory item")
				return
			}
			response.OK(c, item)
		})

		v1.PUT("/:productId", func(c *gin.Context) {
			productID := c.Param("productId")
			var req UpdateStockRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				response.BadRequest(c, err.Error())
				return
			}

			if req.Stock != nil {
				_, err = pool.Exec(c.Request.Context(),
					"UPDATE inventory SET stock = $1, updated_at = NOW() WHERE product_id = $2",
					*req.Stock, productID)
			} else if req.Delta != nil {
				_, err = pool.Exec(c.Request.Context(),
					"UPDATE inventory SET stock = GREATEST(0, stock + $1), updated_at = NOW() WHERE product_id = $2",
					*req.Delta, productID)
			}

			if err != nil {
				response.InternalError(c, "failed to update stock")
				return
			}
			response.OKMessage(c, "stock updated")
		})
	}

	srv := &http.Server{
		Addr:    ":" + viper.GetString("PORT"),
		Handler: router,
	}

	go func() {
		log.Info("inventory service started", zap.String("port", viper.GetString("PORT")))
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

func makeMessageHandler(pool *pgxpool.Pool, producer *pkgkafka.Producer, log *zap.Logger) pkgkafka.MessageHandler {
	return func(msg *sarama.ConsumerMessage) error {
		switch msg.Topic {
		case pkgkafka.TopicOrderCreated:
			var event pkgkafka.OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			return reserveStock(context.Background(), pool, producer, &event, log)

		case pkgkafka.TopicOrderCancelled:
			var event pkgkafka.OrderStatusUpdatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			return releaseStock(context.Background(), pool, event.OrderID, log)
		}
		return nil
	}
}

func reserveStock(ctx context.Context, pool *pgxpool.Pool, producer *pkgkafka.Producer, event *pkgkafka.OrderCreatedEvent, log *zap.Logger) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	allReserved := true
	for _, item := range event.Items {
		var available int
		err := tx.QueryRow(ctx,
			"SELECT (stock - reserved) FROM inventory WHERE product_id = $1 FOR UPDATE",
			item.ProductID,
		).Scan(&available)
		if err != nil || available < item.Quantity {
			allReserved = false
			break
		}

		_, err = tx.Exec(ctx,
			"UPDATE inventory SET reserved = reserved + $1, updated_at = NOW() WHERE product_id = $2",
			item.Quantity, item.ProductID,
		)
		if err != nil {
			allReserved = false
			break
		}
	}

	if !allReserved {
		log.Warn("failed to reserve stock for order", zap.String("order_id", event.OrderID))
		return tx.Rollback(ctx)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if producer != nil {
		reservedEvent := pkgkafka.InventoryReservedEvent{
			OrderID:  event.OrderID,
			Items:    event.Items,
			Reserved: true,
		}
		_ = producer.Publish(pkgkafka.TopicInventoryReserved, event.OrderID, reservedEvent)
	}

	log.Info("stock reserved for order", zap.String("order_id", event.OrderID))
	return nil
}

func releaseStock(ctx context.Context, pool *pgxpool.Pool, orderID string, log *zap.Logger) error {
	_ = uuid.MustParse(orderID)
	log.Info("stock release triggered", zap.String("order_id", orderID))
	return nil
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
