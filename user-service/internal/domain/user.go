package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrForbidden    = errors.New("forbidden")
)

type User struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Email     string    `db:"email"      json:"email"`
	Username  string    `db:"username"   json:"username"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name"  json:"last_name"`
	Phone     string    `db:"phone"      json:"phone,omitempty"`
	Address   string    `db:"address"    json:"address,omitempty"`
	AvatarURL string    `db:"avatar_url" json:"avatar_url,omitempty"`
	Role      string    `db:"role"       json:"role"`
	IsActive  bool      `db:"is_active"  json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

type UserFilter struct {
	Search   string
	Role     string
	IsActive *bool
	Page     int
	Limit    int
}
