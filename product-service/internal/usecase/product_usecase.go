package usecase

import (
	"context"
	"time"

	"github.com/diploma/product-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error)
	Update(ctx context.Context, id string, updates bson.M) error
	Delete(ctx context.Context, id string) error
	UpdateStock(ctx context.Context, id string, delta int) error
	AddImage(ctx context.Context, id, imageURL string) error
}

type SearchRepository interface {
	IndexProduct(ctx context.Context, p *domain.Product) error
	Search(ctx context.Context, req *domain.SearchRequest) ([]string, int64, error)
	DeleteProduct(ctx context.Context, id string) error
	UpdateProduct(ctx context.Context, id string, updates map[string]interface{}) error
}

type StorageRepository interface {
	UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error)
	DeleteFile(ctx context.Context, bucket, objectName string) error
}

type ProductUsecase struct {
	repo    ProductRepository
	search  SearchRepository
	storage StorageRepository
	log     *zap.Logger
}

func NewProductUsecase(repo ProductRepository, search SearchRepository, storage StorageRepository, log *zap.Logger) *ProductUsecase {
	return &ProductUsecase{
		repo:    repo,
		search:  search,
		storage: storage,
		log:     log,
	}
}

func (uc *ProductUsecase) Create(ctx context.Context, req *domain.CreateProductRequest) (*domain.Product, error) {
	if _, err := uc.repo.GetBySKU(ctx, req.SKU); err == nil {
		return nil, domain.ErrProductExists
	}

	p := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		SKU:         req.SKU,
		Stock:       req.Stock,
		Tags:        req.Tags,
		Weight:      req.Weight,
		Images:      []string{},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	if err := uc.search.IndexProduct(ctx, p); err != nil {
		uc.log.Error("failed to index product in elasticsearch", zap.Error(err))
	}

	return p, nil
}

func (uc *ProductUsecase) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ProductUsecase) List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error) {
	return uc.repo.List(ctx, filter)
}

func (uc *ProductUsecase) Search(ctx context.Context, req *domain.SearchRequest) ([]*domain.Product, int64, error) {
	ids, total, err := uc.search.Search(ctx, req)
	if err != nil {
		uc.log.Error("elasticsearch search failed, falling back to mongo", zap.Error(err))
		filter := domain.ProductFilter{
			Category: req.Category,
			MinPrice: req.MinPrice,
			MaxPrice: req.MaxPrice,
			Search:   req.Query,
			Page:     req.Page,
			Limit:    req.Limit,
		}
		active := true
		filter.IsActive = &active
		return uc.repo.List(ctx, filter)
	}

	products := make([]*domain.Product, 0, len(ids))
	for _, id := range ids {
		p, err := uc.repo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		products = append(products, p)
	}

	return products, total, nil
}

func (uc *ProductUsecase) Update(ctx context.Context, id string, req *domain.UpdateProductRequest) (*domain.Product, error) {
	updates := bson.M{}
	esUpdates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
		esUpdates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
		esUpdates["description"] = req.Description
	}
	if req.Price > 0 {
		updates["price"] = req.Price
		esUpdates["price"] = req.Price
	}
	if req.Category != "" {
		updates["category"] = req.Category
		esUpdates["category"] = req.Category
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
		esUpdates["tags"] = req.Tags
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		esUpdates["is_active"] = *req.IsActive
	}
	if req.Weight > 0 {
		updates["weight"] = req.Weight
	}
	if req.Stock >= 0 {
		updates["stock"] = req.Stock
	}

	if err := uc.repo.Update(ctx, id, updates); err != nil {
		return nil, err
	}

	if len(esUpdates) > 0 {
		if err := uc.search.UpdateProduct(ctx, id, esUpdates); err != nil {
			uc.log.Error("failed to update product in elasticsearch", zap.Error(err))
		}
	}

	return uc.repo.GetByID(ctx, id)
}

func (uc *ProductUsecase) Delete(ctx context.Context, id string) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := uc.search.DeleteProduct(ctx, id); err != nil {
		uc.log.Error("failed to delete product from elasticsearch", zap.Error(err))
	}

	return nil
}

func (uc *ProductUsecase) UploadImage(ctx context.Context, productID string, filename string, data []byte, contentType string) (*domain.Product, error) {
	product, err := uc.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	objectName := "products/" + product.ID.Hex() + "/" + filename
	imageURL, err := uc.storage.UploadFile(ctx, "products", objectName, data, contentType)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.AddImage(ctx, productID, imageURL); err != nil {
		return nil, err
	}

	product.Images = append(product.Images, imageURL)
	return product, nil
}
