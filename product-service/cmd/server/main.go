package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diploma/product-service/internal/config"
	httpdelivery "github.com/diploma/product-service/internal/delivery/http"
	elasticrepo "github.com/diploma/product-service/internal/repository/elastic"
	miniorepo "github.com/diploma/product-service/internal/repository/minio"
	mongorepo "github.com/diploma/product-service/internal/repository/mongo"
	"github.com/diploma/product-service/internal/usecase"
	"github.com/diploma/pkg/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URL))
	if err != nil {
		log.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer mongoClient.Disconnect(ctx)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal("mongodb ping failed", zap.Error(err))
	}
	log.Info("connected to mongodb")

	db := mongoClient.Database(cfg.Mongo.Database)

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.Elasticsearch.URL},
	})
	if err != nil {
		log.Fatal("failed to create elasticsearch client", zap.Error(err))
	}

	storageRepo, err := miniorepo.NewStorageRepository(
		cfg.Minio.Endpoint, cfg.Minio.AccessKey, cfg.Minio.SecretKey, cfg.Minio.UseSSL, log,
	)
	if err != nil {
		log.Error("failed to create minio client, images disabled", zap.Error(err))
		storageRepo = nil
	} else {
		if err := storageRepo.EnsureBucket(ctx, cfg.Minio.Bucket); err != nil {
			log.Warn("failed to ensure minio bucket", zap.Error(err))
		} else {
			log.Info("connected to minio")
		}
	}

	productRepo := mongorepo.NewProductRepository(db)
	searchRepo := elasticrepo.NewSearchRepository(esClient, log)
	if err := searchRepo.CreateIndex(ctx); err != nil {
		log.Warn("failed to create elasticsearch index", zap.Error(err))
	}

	productUC := usecase.NewProductUsecase(productRepo, searchRepo, storageRepo, log)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	handler := httpdelivery.NewHandler(productUC, log)
	handler.Register(router)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("product service started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down product service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
	}

	log.Info("product service stopped")
}
