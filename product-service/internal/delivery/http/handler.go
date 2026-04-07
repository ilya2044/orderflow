package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/diploma/product-service/internal/domain"
	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type ProductUsecase interface {
	Create(ctx context.Context, req *domain.CreateProductRequest) (*domain.Product, error)
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error)
	Search(ctx context.Context, req *domain.SearchRequest) ([]*domain.Product, int64, error)
	Update(ctx context.Context, id string, req *domain.UpdateProductRequest) (*domain.Product, error)
	Delete(ctx context.Context, id string) error
	UploadImage(ctx context.Context, productID, filename string, data []byte, contentType string) (*domain.Product, error)
}

type Handler struct {
	usecase ProductUsecase
	log     *zap.Logger
}

func NewHandler(usecase ProductUsecase, log *zap.Logger) *Handler {
	return &Handler{usecase: usecase, log: log}
}

func (h *Handler) Register(router *gin.Engine) {
	router.GET("/health", h.health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/products")
	{
		v1.GET("", h.listProducts)
		v1.GET("/search", h.searchProducts)
		v1.GET("/:id", h.getProduct)
		v1.POST("", h.createProduct)
		v1.PUT("/:id", h.updateProduct)
		v1.DELETE("/:id", h.deleteProduct)
		v1.POST("/:id/images", h.uploadImage)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "product-service", "timestamp": time.Now().UTC()})
}

func (h *Handler) listProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)

	filter := domain.ProductFilter{
		Category: c.Query("category"),
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Search:   c.Query("search"),
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
		Page:     page,
		Limit:    limit,
	}

	isActiveStr := c.Query("is_active")
	if isActiveStr != "" {
		active := isActiveStr == "true"
		filter.IsActive = &active
	} else {
		active := true
		filter.IsActive = &active
	}

	products, total, err := h.usecase.List(c.Request.Context(), filter)
	if err != nil {
		h.log.Error("list products failed", zap.Error(err))
		response.InternalError(c, "failed to get products")
		return
	}

	response.Paginated(c, products, total, page, limit)
}

func (h *Handler) searchProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)

	req := &domain.SearchRequest{
		Query:    c.Query("q"),
		Category: c.Query("category"),
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Page:     page,
		Limit:    limit,
	}

	products, total, err := h.usecase.Search(c.Request.Context(), req)
	if err != nil {
		h.log.Error("search products failed", zap.Error(err))
		response.InternalError(c, "search failed")
		return
	}

	response.Paginated(c, products, total, page, limit)
}

func (h *Handler) getProduct(c *gin.Context) {
	id := c.Param("id")
	product, err := h.usecase.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, "failed to get product")
		return
	}
	response.OK(c, product)
}

func (h *Handler) createProduct(c *gin.Context) {
	var req domain.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	product, err := h.usecase.Create(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrProductExists) {
			response.Conflict(c, "product with this SKU already exists")
			return
		}
		h.log.Error("create product failed", zap.Error(err))
		response.InternalError(c, "failed to create product")
		return
	}

	response.Created(c, product)
}

func (h *Handler) updateProduct(c *gin.Context) {
	id := c.Param("id")

	var req domain.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	product, err := h.usecase.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		h.log.Error("update product failed", zap.Error(err))
		response.InternalError(c, "failed to update product")
		return
	}

	response.OK(c, product)
}

func (h *Handler) deleteProduct(c *gin.Context) {
	id := c.Param("id")

	if err := h.usecase.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		h.log.Error("delete product failed", zap.Error(err))
		response.InternalError(c, "failed to delete product")
		return
	}

	response.OKMessage(c, "product deleted successfully")
}

func (h *Handler) uploadImage(c *gin.Context) {
	id := c.Param("id")

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response.BadRequest(c, "image file required")
		return
	}
	defer file.Close()

	const maxSize = 10 << 20
	if header.Size > maxSize {
		response.BadRequest(c, "image file too large (max 10MB)")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "failed to read file")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	product, err := h.usecase.UploadImage(c.Request.Context(), id, header.Filename, data, contentType)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		h.log.Error("upload image failed", zap.Error(err))
		response.InternalError(c, "failed to upload image")
		return
	}

	response.OK(c, product)
}
