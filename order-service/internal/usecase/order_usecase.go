package usecase

import (
	"context"
	"time"

	"github.com/diploma/order-service/internal/domain"
	pkgkafka "github.com/diploma/pkg/kafka"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByUserID(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error)
	GetAll(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type KafkaProducer interface {
	Publish(topic, key string, payload interface{}) error
}

type OrderUsecase struct {
	repo     OrderRepository
	producer KafkaProducer
	log      *zap.Logger
}

func NewOrderUsecase(repo OrderRepository, producer KafkaProducer, log *zap.Logger) *OrderUsecase {
	return &OrderUsecase{
		repo:     repo,
		producer: producer,
		log:      log,
	}
}

func (uc *OrderUsecase) CreateOrder(ctx context.Context, userID string, req *domain.CreateOrderRequest) (*domain.Order, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	order := &domain.Order{
		ID:              uuid.New(),
		UserID:          uid,
		Status:          domain.StatusPending,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	var total float64
	items := make([]domain.OrderItem, 0, len(req.Items))
	kafkaItems := make([]pkgkafka.OrderItem, 0, len(req.Items))

	for _, reqItem := range req.Items {
		item := domain.OrderItem{
			ID:          uuid.New(),
			OrderID:     order.ID,
			ProductID:   reqItem.ProductID,
			ProductName: reqItem.Name,
			Price:       reqItem.Price,
			Quantity:    reqItem.Quantity,
			CreatedAt:   now,
		}
		total += reqItem.Price * float64(reqItem.Quantity)
		items = append(items, item)
		kafkaItems = append(kafkaItems, pkgkafka.OrderItem{
			ProductID: reqItem.ProductID,
			Quantity:  reqItem.Quantity,
			Price:     reqItem.Price,
		})
	}

	order.TotalPrice = total
	order.Items = items

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	event := pkgkafka.OrderCreatedEvent{
		OrderID:    order.ID.String(),
		UserID:     userID,
		Items:      kafkaItems,
		TotalPrice: total,
		CreatedAt:  now.UTC().String(),
	}

	if err := uc.producer.Publish(pkgkafka.TopicOrderCreated, order.ID.String(), event); err != nil {
		uc.log.Error("failed to publish order.created event",
			zap.String("order_id", order.ID.String()),
			zap.Error(err),
		)
	}

	return order, nil
}

func (uc *OrderUsecase) GetOrder(ctx context.Context, orderID, userID, role string) (*domain.Order, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, domain.ErrOrderNotFound
	}

	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if role != "admin" && order.UserID.String() != userID {
		return nil, domain.ErrOrderNotFound
	}

	return order, nil
}

func (uc *OrderUsecase) GetUserOrders(ctx context.Context, userID string, filter domain.OrderFilter) ([]*domain.Order, int64, error) {
	filter.UserID = userID
	return uc.repo.GetByUserID(ctx, filter)
}

func (uc *OrderUsecase) GetAllOrders(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error) {
	return uc.repo.GetAll(ctx, filter)
}

func (uc *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID string, req *domain.UpdateStatusRequest) (*domain.Order, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, domain.ErrOrderNotFound
	}

	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !order.Status.CanTransitionTo(req.Status) {
		return nil, domain.ErrInvalidStatusTransition
	}

	oldStatus := order.Status
	if err := uc.repo.UpdateStatus(ctx, id, req.Status); err != nil {
		return nil, err
	}

	order.Status = req.Status
	order.UpdatedAt = time.Now()

	event := pkgkafka.OrderStatusUpdatedEvent{
		OrderID:   order.ID.String(),
		UserID:    order.UserID.String(),
		OldStatus: string(oldStatus),
		NewStatus: string(req.Status),
		UpdatedAt: order.UpdatedAt.UTC().String(),
	}

	if err := uc.producer.Publish(pkgkafka.TopicOrderStatusUpdated, order.ID.String(), event); err != nil {
		uc.log.Error("failed to publish order.status_updated event",
			zap.String("order_id", order.ID.String()),
			zap.Error(err),
		)
	}

	return order, nil
}

func (uc *OrderUsecase) CancelOrder(ctx context.Context, orderID, userID, role string) error {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return domain.ErrOrderNotFound
	}

	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if role != "admin" && order.UserID.String() != userID {
		return domain.ErrOrderNotFound
	}

	if order.Status == domain.StatusCancelled {
		return domain.ErrOrderAlreadyCancelled
	}

	if !order.Status.CanTransitionTo(domain.StatusCancelled) {
		return domain.ErrOrderNotCancellable
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return err
	}

	event := pkgkafka.OrderStatusUpdatedEvent{
		OrderID:   order.ID.String(),
		UserID:    order.UserID.String(),
		OldStatus: string(order.Status),
		NewStatus: string(domain.StatusCancelled),
		UpdatedAt: time.Now().UTC().String(),
	}

	if err := uc.producer.Publish(pkgkafka.TopicOrderCancelled, order.ID.String(), event); err != nil {
		uc.log.Error("failed to publish order.cancelled event", zap.Error(err))
	}

	return nil
}
