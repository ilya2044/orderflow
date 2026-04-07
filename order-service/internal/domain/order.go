package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyCancelled = errors.New("order already cancelled")
	ErrOrderNotCancellable   = errors.New("order cannot be cancelled in current status")
	ErrInsufficientStock     = errors.New("insufficient stock")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusConfirmed  OrderStatus = "confirmed"
	StatusProcessing OrderStatus = "processing"
	StatusShipped    OrderStatus = "shipped"
	StatusDelivered  OrderStatus = "delivered"
	StatusCancelled  OrderStatus = "cancelled"
	StatusRefunded   OrderStatus = "refunded"
)

var validTransitions = map[OrderStatus][]OrderStatus{
	StatusPending:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed:  {StatusProcessing, StatusCancelled},
	StatusProcessing: {StatusShipped, StatusCancelled},
	StatusShipped:    {StatusDelivered},
	StatusDelivered:  {StatusRefunded},
	StatusCancelled:  {},
	StatusRefunded:   {},
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

type Order struct {
	ID              uuid.UUID   `db:"id"               json:"id"`
	UserID          uuid.UUID   `db:"user_id"          json:"user_id"`
	Status          OrderStatus `db:"status"           json:"status"`
	TotalPrice      float64     `db:"total_price"      json:"total_price"`
	ShippingAddress string      `db:"shipping_address" json:"shipping_address"`
	Notes           string      `db:"notes"            json:"notes,omitempty"`
	CreatedAt       time.Time   `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at"       json:"updated_at"`
	Items           []OrderItem `db:"-"                json:"items,omitempty"`
}

type OrderItem struct {
	ID          uuid.UUID `db:"id"           json:"id"`
	OrderID     uuid.UUID `db:"order_id"     json:"order_id"`
	ProductID   string    `db:"product_id"   json:"product_id"`
	ProductName string    `db:"product_name" json:"product_name"`
	Price       float64   `db:"price"        json:"price"`
	Quantity    int       `db:"quantity"     json:"quantity"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
}

type CreateOrderRequest struct {
	ShippingAddress string              `json:"shipping_address" binding:"required"`
	Notes           string              `json:"notes"`
	Items           []CreateOrderItem   `json:"items"           binding:"required,min=1,dive"`
}

type CreateOrderItem struct {
	ProductID string  `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity"   binding:"required,min=1"`
	Price     float64 `json:"price"      binding:"required,gt=0"`
	Name      string  `json:"name"       binding:"required"`
}

type UpdateStatusRequest struct {
	Status OrderStatus `json:"status" binding:"required"`
}

type OrderFilter struct {
	UserID string
	Status OrderStatus
	Page   int
	Limit  int
}
