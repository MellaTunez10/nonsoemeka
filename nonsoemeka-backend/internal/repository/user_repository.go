package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
	"time"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type UserRepository interface {
	Create(ctx context.Context, db DBTX, user models.User) (models.User, error)
	FindByUsername(ctx context.Context, db DBTX, username string) (models.User, error)
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.User, error)
	Update(ctx context.Context, db DBTX, user models.User) error
	IncrementFailedLogin(ctx context.Context, db DBTX, userID uuid.UUID) (int, error)
	ResetFailedLogin(ctx context.Context, db DBTX, userID uuid.UUID) error
	LockUser(ctx context.Context, db DBTX, userID uuid.UUID, until time.Time) error
	List(ctx context.Context, db DBTX, page, pageSize int) ([]models.User, int, error)
	SaveRefreshToken(ctx context.Context, db DBTX, token models.RefreshToken) error
	FindRefreshToken(ctx context.Context, db DBTX, tokenHash string) (models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, db DBTX, tokenID uuid.UUID) error
	RevokeAllUserRefreshTokens(ctx context.Context, db DBTX, userID uuid.UUID) error
	Delete(ctx context.Context, db DBTX, id uuid.UUID) error
}

type postgresUserRepository struct{}

func NewUserRepository() UserRepository {
	return &postgresUserRepository{}
}

func (r *postgresUserRepository) Create(ctx context.Context, db DBTX, user models.User) (models.User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password_hash, role, is_active, failed_login_attempts, locked_until, created_at, updated_at
	`
	var created models.User
	err := db.QueryRow(ctx, query, user.Username, user.Email, user.PasswordHash, user.Role, user.IsActive).Scan(
		&created.ID, &created.Username, &created.Email, &created.PasswordHash, &created.Role,
		&created.IsActive, &created.FailedLoginAttempts, &created.LockedUntil, &created.CreatedAt, &created.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, fmt.Errorf("user already exists: %w", apperrors.ErrBadRequest)
		}
		return models.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	return created, nil
}

func (r *postgresUserRepository) FindByUsername(ctx context.Context, db DBTX, username string) (models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, failed_login_attempts, locked_until, created_at, updated_at
		FROM users WHERE username = $1
	`
	var u models.User
	err := db.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.IsActive, &u.FailedLoginAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, apperrors.ErrInvalidCredentials
		}
		return models.User{}, fmt.Errorf("failed to query user by username: %w", err)
	}

	return u, nil
}

func (r *postgresUserRepository) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, failed_login_attempts, locked_until, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u models.User
	err := db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.IsActive, &u.FailedLoginAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, apperrors.ErrNotFound
		}
		return models.User{}, fmt.Errorf("failed to query user by id: %w", err)
	}

	return u, nil
}

func (r *postgresUserRepository) Update(ctx context.Context, db DBTX, u models.User) error {
	query := `
		UPDATE users
		SET is_active = $1, password_hash = $2, locked_until = $3, updated_at = now()
		WHERE id = $4
	`
	cmd, err := db.Exec(ctx, query, u.IsActive, u.PasswordHash, u.LockedUntil, u.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *postgresUserRepository) IncrementFailedLogin(ctx context.Context, db DBTX, userID uuid.UUID) (int, error) {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1, updated_at = now()
		WHERE id = $1
		RETURNING failed_login_attempts
	`
	var attempts int
	err := db.QueryRow(ctx, query, userID).Scan(&attempts)
	if err != nil {
		return 0, fmt.Errorf("failed to increment failed login: %w", err)
	}
	return attempts, nil
}

func (r *postgresUserRepository) ResetFailedLogin(ctx context.Context, db DBTX, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = now()
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to reset failed login: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) LockUser(ctx context.Context, db DBTX, userID uuid.UUID, until time.Time) error {
	query := `
		UPDATE users
		SET locked_until = $1, updated_at = now()
		WHERE id = $2
	`
	_, err := db.Exec(ctx, query, until, userID)
	if err != nil {
		return fmt.Errorf("failed to lock user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) List(ctx context.Context, db DBTX, page, pageSize int) ([]models.User, int, error) {
	offset := (page - 1) * pageSize
	countQuery := `SELECT COUNT(*) FROM users`
	var total int
	if err := db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT id, username, email, password_hash, role, is_active, failed_login_attempts, locked_until, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
			&u.IsActive, &u.FailedLoginAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *postgresUserRepository) SaveRefreshToken(ctx context.Context, db DBTX, token models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return db.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt).Scan(&token.ID, &token.CreatedAt)
}

func (r *postgresUserRepository) FindRefreshToken(ctx context.Context, db DBTX, tokenHash string) (models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var t models.RefreshToken
	err := db.QueryRow(ctx, query, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshToken{}, apperrors.ErrUnauthorized
		}
		return models.RefreshToken{}, fmt.Errorf("failed to query refresh token: %w", err)
	}
	return t, nil
}

func (r *postgresUserRepository) RevokeRefreshToken(ctx context.Context, db DBTX, tokenID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1`
	_, err := db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) RevokeAllUserRefreshTokens(ctx context.Context, db DBTX, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) Delete(ctx context.Context, db DBTX, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	cmd, err := db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}
