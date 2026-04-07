package kafka

const (
	TopicOrderCreated        = "order.created"
	TopicOrderStatusUpdated  = "order.status_updated"
	TopicOrderCancelled      = "order.cancelled"
	TopicPaymentProcessed    = "payment.processed"
	TopicPaymentFailed       = "payment.failed"
	TopicInventoryReserved   = "inventory.reserved"
	TopicInventoryReleased   = "inventory.released"
	TopicNotificationSend    = "notification.send"
)

type OrderCreatedEvent struct {
	OrderID     string      `json:"order_id"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items"`
	TotalPrice  float64     `json:"total_price"`
	CreatedAt   string      `json:"created_at"`
}

type OrderStatusUpdatedEvent struct {
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	UpdatedAt string `json:"updated_at"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type PaymentProcessedEvent struct {
	PaymentID string  `json:"payment_id"`
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Method    string  `json:"method"`
}

type InventoryReservedEvent struct {
	OrderID   string      `json:"order_id"`
	Items     []OrderItem `json:"items"`
	Reserved  bool        `json:"reserved"`
}

type NotificationEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Channel   string `json:"channel"`
}
