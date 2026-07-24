package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/audit"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type StaffManagementService interface {
	CreateStaff(ctx context.Context, actorID uuid.UUID, req dto.CreateStaffRequest) (dto.StaffResponse, error)
	ListStaff(ctx context.Context, page, pageSize int) (dto.PaginatedResponse[dto.StaffResponse], error)
	UpdateStaff(ctx context.Context, actorID uuid.UUID, staffID uuid.UUID, req dto.UpdateStaffRequest) (dto.StaffResponse, error)
	DeleteStaff(ctx context.Context, actorID uuid.UUID, staffID uuid.UUID) error
	ListAuditLogs(ctx context.Context, actorID *uuid.UUID, action, targetTable *string, startDate, endDate *string, page, pageSize int) (dto.PaginatedResponse[dto.AuditLogResponse], error)
}

type staffManagementService struct {
	pool      *pgxpool.Pool
	userRepo  repository.UserRepository
	auditRepo repository.AuditRepository
}

func NewStaffManagementService(pool *pgxpool.Pool, userRepo repository.UserRepository, auditRepo repository.AuditRepository) StaffManagementService {
	return &staffManagementService{
		pool:      pool,
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

func (s *staffManagementService) CreateStaff(ctx context.Context, actorID uuid.UUID, req dto.CreateStaffRequest) (dto.StaffResponse, error) {
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return dto.StaffResponse{}, err
	}

	role := models.UserRole(req.Role)
	if role != models.RoleAdmin && role != models.RoleStaff {
		role = models.RoleStaff
	}

	userModel := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         role,
		IsActive:     true,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.StaffResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	createdUser, err := s.userRepo.Create(ctx, tx, userModel)
	if err != nil {
		return dto.StaffResponse{}, err
	}

	if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "USER_CREATED", "users", &createdUser.ID, map[string]interface{}{
		"username": createdUser.Username,
		"email":    createdUser.Email,
		"role":     string(createdUser.Role),
	}); err != nil {
		return dto.StaffResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.StaffResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dto.StaffResponse{
		ID:                  createdUser.ID,
		Username:            createdUser.Username,
		Email:               createdUser.Email,
		Role:                string(createdUser.Role),
		IsActive:            createdUser.IsActive,
		FailedLoginAttempts: createdUser.FailedLoginAttempts,
		LockedUntil:         createdUser.LockedUntil,
		CreatedAt:           createdUser.CreatedAt,
		UpdatedAt:           createdUser.UpdatedAt,
	}, nil
}

func (s *staffManagementService) ListStaff(ctx context.Context, page, pageSize int) (dto.PaginatedResponse[dto.StaffResponse], error) {
	users, total, err := s.userRepo.List(ctx, s.pool, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.StaffResponse]{}, err
	}

	resList := make([]dto.StaffResponse, 0, len(users))
	for _, u := range users {
		resList = append(resList, dto.StaffResponse{
			ID:                  u.ID,
			Username:            u.Username,
			Email:               u.Email,
			Role:                string(u.Role),
			IsActive:            u.IsActive,
			FailedLoginAttempts: u.FailedLoginAttempts,
			LockedUntil:         u.LockedUntil,
			CreatedAt:           u.CreatedAt,
			UpdatedAt:           u.UpdatedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.StaffResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *staffManagementService) UpdateStaff(ctx context.Context, actorID uuid.UUID, staffID uuid.UUID, req dto.UpdateStaffRequest) (dto.StaffResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.StaffResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := s.userRepo.FindByID(ctx, tx, staffID)
	if err != nil {
		return dto.StaffResponse{}, err
	}

	changes := make(map[string]interface{})

	if req.IsActive != nil {
		changes["is_active"] = map[string]bool{"before": user.IsActive, "after": *req.IsActive}
		user.IsActive = *req.IsActive
	}

	if req.Password != nil && *req.Password != "" {
		newHash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return dto.StaffResponse{}, err
		}
		user.PasswordHash = newHash
		changes["password"] = "RESET"
	}

	if req.ClearLockout {
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		changes["lockout"] = "CLEARED"
	}

	if err := s.userRepo.Update(ctx, tx, user); err != nil {
		return dto.StaffResponse{}, err
	}

	if len(changes) > 0 {
		if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "USER_UPDATED", "users", &user.ID, changes); err != nil {
			return dto.StaffResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.StaffResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dto.StaffResponse{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               user.Email,
		Role:                string(user.Role),
		IsActive:            user.IsActive,
		FailedLoginAttempts: user.FailedLoginAttempts,
		LockedUntil:         user.LockedUntil,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}, nil
}

func (s *staffManagementService) ListAuditLogs(ctx context.Context, actorID *uuid.UUID, action, targetTable *string, startDate, endDate *string, page, pageSize int) (dto.PaginatedResponse[dto.AuditLogResponse], error) {
	logs, total, err := s.auditRepo.List(ctx, s.pool, actorID, action, targetTable, startDate, endDate, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.AuditLogResponse]{}, err
	}

	resList := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		resList = append(resList, dto.AuditLogResponse{
			ID:          l.ID,
			ActorID:     l.ActorID,
			ActorName:   l.ActorName,
			Action:      l.Action,
			TargetTable: l.TargetTable,
			TargetID:    l.TargetID,
			Metadata:    l.Metadata,
			CreatedAt:   l.CreatedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.AuditLogResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *staffManagementService) DeleteStaff(ctx context.Context, actorID uuid.UUID, staffID uuid.UUID) error {
	if actorID == staffID {
		return fmt.Errorf("cannot delete your own account: %w", apperrors.ErrBadRequest)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := s.userRepo.FindByID(ctx, tx, staffID)
	if err != nil {
		return err
	}

	if err := s.userRepo.RevokeAllUserRefreshTokens(ctx, tx, staffID); err != nil {
		return err
	}

	if err := s.userRepo.Delete(ctx, tx, staffID); err != nil {
		return err
	}

	if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "USER_DELETED", "users", &staffID, map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"role":     string(user.Role),
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
