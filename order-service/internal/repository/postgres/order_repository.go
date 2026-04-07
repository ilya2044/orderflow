package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diploma/order-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO orders (id, user_id, status, total_price, shipping_address, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, query,
		order.ID, order.UserID, order.Status, order.TotalPrice,
		order.ShippingAddress, order.Notes, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return err
	}

	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, product_name, price, quantity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for i := range order.Items {
		item := &order.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now()
		}
		_, err = tx.Exec(ctx, itemQuery,
			item.ID, order.ID, item.ProductID, item.ProductName,
			item.Price, item.Quantity, item.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	query := `
		SELECT id, user_id, status, total_price, shipping_address, notes, created_at, updated_at
		FROM orders WHERE id = $1
	`
	order := &domain.Order{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.Status, &order.TotalPrice,
		&order.ShippingAddress, &order.Notes, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	items, err := r.getItems(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return order, nil
}

func (r *OrderRepository) GetByUserID(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error) {
	baseQuery := "FROM orders WHERE user_id = $1"
	args := []interface{}{filter.UserID}
	argIdx := 2

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	offset := (filter.Page - 1) * filter.Limit
	dataQuery := fmt.Sprintf(
		"SELECT id, user_id, status, total_price, shipping_address, notes, created_at, updated_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		baseQuery, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Status, &o.TotalPrice,
			&o.ShippingAddress, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}

	return orders, total, nil
}

func (r *OrderRepository) GetAll(ctx context.Context, filter domain.OrderFilter) ([]*domain.Order, int64, error) {
	baseQuery := "FROM orders WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	offset := (filter.Page - 1) * filter.Limit
	dataQuery := fmt.Sprintf(
		"SELECT id, user_id, status, total_price, shipping_address, notes, created_at, updated_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		baseQuery, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Status, &o.TotalPrice,
			&o.ShippingAddress, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}

	return orders, total, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3",
		status, time.Now(), id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (r *OrderRepository) getItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, order_id, product_id, product_name, price, quantity, created_at FROM order_items WHERE order_id = $1",
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		item := domain.OrderItem{}
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.ProductName,
			&item.Price, &item.Quantity, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
