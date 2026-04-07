package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrProductExists   = errors.New("product with this SKU already exists")
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	Name        string             `bson:"name"            json:"name"`
	Description string             `bson:"description"     json:"description"`
	Price       float64            `bson:"price"           json:"price"`
	Category    string             `bson:"category"        json:"category"`
	SKU         string             `bson:"sku"             json:"sku"`
	Stock       int                `bson:"stock"           json:"stock"`
	Images      []string           `bson:"images"          json:"images,omitempty"`
	Tags        []string           `bson:"tags"            json:"tags,omitempty"`
	IsActive    bool               `bson:"is_active"       json:"is_active"`
	Rating      float64            `bson:"rating"          json:"rating"`
	ReviewCount int                `bson:"review_count"    json:"review_count"`
	Weight      float64            `bson:"weight"          json:"weight,omitempty"`
	CreatedAt   time.Time          `bson:"created_at"      json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"      json:"updated_at"`
}

type CreateProductRequest struct {
	Name        string   `json:"name"        binding:"required,min=2,max=255"`
	Description string   `json:"description" binding:"required"`
	Price       float64  `json:"price"       binding:"required,gt=0"`
	Category    string   `json:"category"    binding:"required"`
	SKU         string   `json:"sku"         binding:"required"`
	Stock       int      `json:"stock"       binding:"min=0"`
	Tags        []string `json:"tags"`
	Weight      float64  `json:"weight"`
}

type UpdateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Category    string   `json:"category"`
	Stock       int      `json:"stock"`
	Tags        []string `json:"tags"`
	IsActive    *bool    `json:"is_active"`
	Weight      float64  `json:"weight"`
}

type ProductFilter struct {
	Category string
	MinPrice float64
	MaxPrice float64
	Tags     []string
	IsActive *bool
	Search   string
	Page     int
	Limit    int
	SortBy   string
	SortDir  string
}

type SearchRequest struct {
	Query    string  `form:"q"`
	Category string  `form:"category"`
	MinPrice float64 `form:"min_price"`
	MaxPrice float64 `form:"max_price"`
	Page     int     `form:"page"`
	Limit    int     `form:"limit"`
}
