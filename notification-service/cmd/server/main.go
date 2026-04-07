package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pkgkafka "github.com/diploma/pkg/kafka"
	"github.com/diploma/pkg/logger"
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type NotificationRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Channel   string    `json:"channel"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8087")
	viper.SetDefault("LOG_LEVEL", "info")

	log, _ := logger.New(viper.GetString("LOG_LEVEL"))
	defer log.Sync()

	opt, err := redis.ParseURL(viper.GetString("REDIS_URL"))
	if err != nil {
		log.Fatal("failed to parse redis URL", zap.Error(err))
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn("redis unavailable", zap.Error(err))
	} else {
		log.Info("connected to redis")
	}

	brokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	topics := []string{
		pkgkafka.TopicOrderCreated,
		pkgkafka.TopicOrderStatusUpdated,
		pkgkafka.TopicPaymentProcessed,
		pkgkafka.TopicNotificationSend,
	}

	consumer, err := pkgkafka.NewConsumer(brokers, "notification-service", topics,
		makeNotificationHandler(rdb, log), log)
	if err != nil {
		log.Error("kafka consumer unavailable", zap.Error(err))
	} else {
		go func() {
			if err := consumer.Start(context.Background()); err != nil {
				log.Error("consumer error", zap.Error(err))
			}
		}()
		defer consumer.Close()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "notification-service"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/api/v1/notifications/:userId", func(c *gin.Context) {
		userID := c.Param("userId")
		key := fmt.Sprintf("notifications:%s", userID)

		items, err := rdb.LRange(c.Request.Context(), key, 0, 49).Result()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
			return
		}

		notifications := make([]interface{}, 0, len(items))
		for _, item := range items {
			var n NotificationRecord
			if json.Unmarshal([]byte(item), &n) == nil {
				notifications = append(notifications, n)
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": notifications})
	})

	srv := &http.Server{
		Addr:    ":" + viper.GetString("PORT"),
		Handler: router,
	}

	go func() {
		log.Info("notification service started", zap.String("port", viper.GetString("PORT")))
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

func makeNotificationHandler(rdb *redis.Client, log *zap.Logger) pkgkafka.MessageHandler {
	return func(msg *sarama.ConsumerMessage) error {
		var notification *NotificationRecord

		switch msg.Topic {
		case pkgkafka.TopicOrderCreated:
			var event pkgkafka.OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			notification = &NotificationRecord{
				UserID:  event.UserID,
				Type:    "order_created",
				Subject: "Ваш заказ оформлен",
				Body:    fmt.Sprintf("Заказ #%s на сумму %.2f ₽ успешно оформлен и ожидает подтверждения.", event.OrderID[:8], event.TotalPrice),
				Channel: "email",
			}

		case pkgkafka.TopicOrderStatusUpdated:
			var event pkgkafka.OrderStatusUpdatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			notification = &NotificationRecord{
				UserID:  event.UserID,
				Type:    "order_status_updated",
				Subject: "Статус заказа обновлён",
				Body:    fmt.Sprintf("Статус заказа #%s изменён: %s → %s", event.OrderID[:8], event.OldStatus, event.NewStatus),
				Channel: "push",
			}

		case pkgkafka.TopicPaymentProcessed:
			var event pkgkafka.PaymentProcessedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			notification = &NotificationRecord{
				UserID:  event.UserID,
				Type:    "payment_processed",
				Subject: "Оплата прошла успешно",
				Body:    fmt.Sprintf("Платёж на сумму %.2f ₽ по заказу #%s успешно обработан.", event.Amount, event.OrderID[:8]),
				Channel: "email",
			}

		case pkgkafka.TopicNotificationSend:
			var event pkgkafka.NotificationEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			notification = &NotificationRecord{
				UserID:  event.UserID,
				Type:    event.Type,
				Subject: event.Subject,
				Body:    event.Body,
				Channel: event.Channel,
			}
		}

		if notification == nil {
			return nil
		}

		notification.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		notification.Status = "sent"
		notification.CreatedAt = time.Now()

		log.Info("notification sent",
			zap.String("user_id", notification.UserID),
			zap.String("type", notification.Type),
			zap.String("channel", notification.Channel),
		)

		data, err := json.Marshal(notification)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("notifications:%s", notification.UserID)
		pipe := rdb.Pipeline()
		pipe.LPush(context.Background(), key, data)
		pipe.LTrim(context.Background(), key, 0, 99)
		pipe.Expire(context.Background(), key, 30*24*time.Hour)
		_, err = pipe.Exec(context.Background())
		return err
	}
}
