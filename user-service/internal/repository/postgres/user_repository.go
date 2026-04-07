package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diploma/user-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Upsert(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, username, first_name, last_name, phone, address, avatar_url, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			username = EXCLUDED.username,
			role = EXCLUDED.role,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Username, user.FirstName, user.LastName,
		user.Phone, user.Address, user.AvatarURL, user.Role, user.IsActive,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, username, first_name, last_name, phone, address, avatar_url, role, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
		&user.Phone, &user.Address, &user.AvatarURL, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return user, err
}

func (r *UserRepository) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error) {
	baseQuery := "FROM users WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Role != "" {
		baseQuery += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, filter.Role)
		argIdx++
	}
	if filter.IsActive != nil {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *filter.IsActive)
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
		"SELECT id, email, username, first_name, last_name, phone, address, avatar_url, role, is_active, created_at, updated_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		baseQuery, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.FirstName, &u.LastName,
			&u.Phone, &u.Address, &u.AvatarURL, &u.Role, &u.IsActive,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, req *domain.UpdateUserRequest) (*domain.User, error) {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, phone = $3, address = $4, updated_at = $5
		WHERE id = $6
	`
	result, err := r.pool.Exec(ctx, query,
		req.FirstName, req.LastName, req.Phone, req.Address, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		return nil, domain.ErrUserNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE users SET is_active = $1, updated_at = $2 WHERE id = $3",
		active, time.Now(), id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
