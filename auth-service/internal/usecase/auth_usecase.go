package usecase

import (
	"context"
	"time"

	"github.com/diploma/auth-service/internal/domain"
	pkgjwt "github.com/diploma/pkg/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type AuthUsecase struct {
	userRepo    UserRepository
	tokenRepo   RefreshTokenRepository
	jwtManager  *pkgjwt.Manager
}

func NewAuthUsecase(userRepo UserRepository, tokenRepo RefreshTokenRepository, jwtManager *pkgjwt.Manager) *AuthUsecase {
	return &AuthUsecase{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtManager: jwtManager,
	}
}

func (uc *AuthUsecase) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.TokenPair, error) {
	emailExists, err := uc.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, domain.ErrUserAlreadyExists
	}

	usernameExists, err := uc.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if usernameExists {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return uc.generateTokenPair(ctx, user)
}

func (uc *AuthUsecase) Login(ctx context.Context, req *domain.LoginRequest) (*domain.TokenPair, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return uc.generateTokenPair(ctx, user)
}

func (uc *AuthUsecase) Refresh(ctx context.Context, req *domain.RefreshRequest) (*domain.TokenPair, error) {
	rt, err := uc.tokenRepo.GetByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().After(rt.ExpiresAt) {
		_ = uc.tokenRepo.DeleteByToken(ctx, req.RefreshToken)
		return nil, domain.ErrTokenExpired
	}

	user, err := uc.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if err := uc.tokenRepo.DeleteByToken(ctx, req.RefreshToken); err != nil {
		return nil, err
	}

	return uc.generateTokenPair(ctx, user)
}

func (uc *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	return uc.tokenRepo.DeleteByToken(ctx, refreshToken)
}

func (uc *AuthUsecase) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return uc.tokenRepo.DeleteByUserID(ctx, userID)
}

func (uc *AuthUsecase) ValidateToken(tokenStr string) (*pkgjwt.Claims, error) {
	return uc.jwtManager.ParseToken(tokenStr)
}

func (uc *AuthUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, userID)
}

func (uc *AuthUsecase) generateTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	accessToken, err := uc.jwtManager.GenerateAccessToken(
		user.ID.String(), user.Email, string(user.Role),
	)
	if err != nil {
		return nil, err
	}

	refreshTokenStr, expiresAt, err := uc.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := uc.tokenRepo.Create(ctx, rt); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    expiresAt,
		User:         user.ToDTO(),
	}, nil
}
