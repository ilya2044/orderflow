package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/diploma/order-service/internal/domain"
	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type OrderUsecase interface {
	CreateOrder(ctx context.Context, userID string, req *domain.CreateOrderRequest) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID, userID, role string) (*domain.Order, error)
	GetUserOrders(ctx context.Context, userID string, filter domain.OrderFilter) ([]*domain.Order, int64, error)
	GetAllOrders(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error)
	UpdateOrderStatus(ctx context.Context, orderID string, req *domain.UpdateStatusRequest) (*domain.Order, error)
	CancelOrder(ctx context.Context, orderID, userID, role string) error
}

type Handler struct {
	usecase OrderUsecase
	log     *zap.Logger
}

func NewHandler(usecase OrderUsecase, log *zap.Logger) *Handler {
	return &Handler{usecase: usecase, log: log}
}

func (h *Handler) Register(router *gin.Engine) {
	router.GET("/health", h.health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1/orders")
	{
		v1.POST("", h.createOrder)
		v1.GET("", h.listOrders)
		v1.GET("/:id", h.getOrder)
		v1.PUT("/:id/status", h.updateStatus)
		v1.DELETE("/:id", h.cancelOrder)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "order-service", "timestamp": time.Now().UTC()})
}

func (h *Handler) createOrder(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	var req domain.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	order, err := h.usecase.CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		h.log.Error("create order failed", zap.Error(err))
		response.InternalError(c, "failed to create order")
		return
	}

	response.Created(c, order)
}

func (h *Handler) listOrders(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	statusFilter := domain.OrderStatus(c.Query("status"))

	filter := domain.OrderFilter{
		Status: statusFilter,
		Page:   page,
		Limit:  limit,
	}

	var orders []*domain.Order
	var total int64
	var err error

	if role == "admin" {
		orders, total, err = h.usecase.GetAllOrders(c.Request.Context(), filter)
	} else {
		orders, total, err = h.usecase.GetUserOrders(c.Request.Context(), userID, filter)
	}

	if err != nil {
		h.log.Error("list orders failed", zap.Error(err))
		response.InternalError(c, "failed to get orders")
		return
	}

	if orders == nil {
		orders = []*domain.Order{}
	}

	response.Paginated(c, orders, total, page, limit)
}

func (h *Handler) getOrder(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")
	orderID := c.Param("id")

	order, err := h.usecase.GetOrder(c.Request.Context(), orderID, userID, role)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			response.NotFound(c, "order not found")
			return
		}
		response.InternalError(c, "failed to get order")
		return
	}

	response.OK(c, order)
}

func (h *Handler) updateStatus(c *gin.Context) {
	orderID := c.Param("id")
	if _, err := uuid.Parse(orderID); err != nil {
		response.BadRequest(c, "invalid order id")
		return
	}

	var req domain.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	order, err := h.usecase.UpdateOrderStatus(c.Request.Context(), orderID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			response.NotFound(c, "order not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidStatusTransition) {
			response.UnprocessableEntity(c, "invalid status transition")
			return
		}
		h.log.Error("update status failed", zap.Error(err))
		response.InternalError(c, "failed to update order status")
		return
	}

	response.OK(c, order)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")
	orderID := c.Param("id")

	if err := h.usecase.CancelOrder(c.Request.Context(), orderID, userID, role); err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			response.NotFound(c, "order not found")
			return
		}
		if errors.Is(err, domain.ErrOrderAlreadyCancelled) {
			response.Conflict(c, "order is already cancelled")
			return
		}
		if errors.Is(err, domain.ErrOrderNotCancellable) {
			response.UnprocessableEntity(c, "order cannot be cancelled in current status")
			return
		}
		h.log.Error("cancel order failed", zap.Error(err))
		response.InternalError(c, "failed to cancel order")
		return
	}

	response.OKMessage(c, "order cancelled successfully")
}
