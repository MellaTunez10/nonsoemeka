package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (user models.User, accessToken, rawRefreshToken string, err error)
	Refresh(ctx context.Context, rawRefreshToken string) (accessToken, newRawRefreshToken string, err error)
	Logout(ctx context.Context, rawRefreshToken string) error
}

type authService struct {
	pool     *pgxpool.Pool
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(pool *pgxpool.Pool, userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		pool:     pool,
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) Login(ctx context.Context, username, password string) (models.User, string, string, error) {
	user, err := s.userRepo.FindByUsername(ctx, s.pool, username)
	if err != nil {
		return models.User{}, "", "", err
	}

	if !user.IsActive {
		return models.User{}, "", "", apperrors.ErrForbidden
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return models.User{}, "", "", apperrors.ErrUserLocked
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		attempts, incErr := s.userRepo.IncrementFailedLogin(ctx, s.pool, user.ID)
		if incErr == nil && attempts >= 5 {
			lockoutTime := time.Now().Add(15 * time.Minute)
			_ = s.userRepo.LockUser(ctx, s.pool, user.ID, lockoutTime)
			return models.User{}, "", "", apperrors.ErrUserLocked
		}
		return models.User{}, "", "", apperrors.ErrInvalidCredentials
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		_ = s.userRepo.ResetFailedLogin(ctx, s.pool, user.ID)
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, user.Username, string(user.Role),
		s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return models.User{}, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	rawRefreshToken := auth.GenerateRawRefreshToken()
	tokenHash := auth.HashRefreshToken(rawRefreshToken)

	refreshTokenModel := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL),
	}

	if err := s.userRepo.SaveRefreshToken(ctx, s.pool, refreshTokenModel); err != nil {
		return models.User{}, "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return user, accessToken, rawRefreshToken, nil
}

func (s *authService) Refresh(ctx context.Context, rawRefreshToken string) (string, string, error) {
	if rawRefreshToken == "" {
		return "", "", apperrors.ErrUnauthorized
	}

	tokenHash := auth.HashRefreshToken(rawRefreshToken)
	tokenModel, err := s.userRepo.FindRefreshToken(ctx, s.pool, tokenHash)
	if err != nil {
		return "", "", err
	}

	// Reuse detection: if token is already revoked, revoke ALL user refresh tokens immediately!
	if tokenModel.RevokedAt != nil {
		_ = s.userRepo.RevokeAllUserRefreshTokens(ctx, s.pool, tokenModel.UserID)
		return "", "", apperrors.ErrUnauthorized
	}

	if tokenModel.ExpiresAt.Before(time.Now()) {
		return "", "", apperrors.ErrUnauthorized
	}

	user, err := s.userRepo.FindByID(ctx, s.pool, tokenModel.UserID)
	if err != nil {
		return "", "", err
	}

	if !user.IsActive {
		return "", "", apperrors.ErrForbidden
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return "", "", apperrors.ErrUserLocked
	}

	// Revoke the presented refresh token
	if err := s.userRepo.RevokeRefreshToken(ctx, s.pool, tokenModel.ID); err != nil {
		return "", "", fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Issue new access & refresh tokens
	accessToken, err := auth.GenerateAccessToken(
		user.ID, user.Username, string(user.Role),
		s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRawRefreshToken := auth.GenerateRawRefreshToken()
	newTokenHash := auth.HashRefreshToken(newRawRefreshToken)

	newRefreshTokenModel := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL),
	}

	if err := s.userRepo.SaveRefreshToken(ctx, s.pool, newRefreshTokenModel); err != nil {
		return "", "", fmt.Errorf("failed to save new refresh token: %w", err)
	}

	return accessToken, newRawRefreshToken, nil
}

func (s *authService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	tokenHash := auth.HashRefreshToken(rawRefreshToken)
	tokenModel, err := s.userRepo.FindRefreshToken(ctx, s.pool, tokenHash)
	if err != nil {
		return nil // idempotent logout
	}

	return s.userRepo.RevokeRefreshToken(ctx, s.pool, tokenModel.ID)
}
