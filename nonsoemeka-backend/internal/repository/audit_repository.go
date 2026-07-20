package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"nonsoemeka-backend/internal/models"
)

type AuditRepository interface {
	Create(ctx context.Context, tx pgx.Tx, log models.AuditLog) error
	List(ctx context.Context, db DBTX, actorID *uuid.UUID, action, targetTable *string, startDate, endDate *string, page, pageSize int) ([]models.AuditLog, int, error)
}

type postgresAuditRepository struct{}

func NewAuditRepository() AuditRepository {
	return &postgresAuditRepository{}
}

func (r *postgresAuditRepository) Create(ctx context.Context, tx pgx.Tx, log models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (actor_id, action, target_table, target_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, log.ActorID, log.Action, log.TargetTable, log.TargetID, log.Metadata)
	} else {
		return fmt.Errorf("transaction required for audit log insertion")
	}
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

func (r *postgresAuditRepository) List(ctx context.Context, db DBTX, actorID *uuid.UUID, action, targetTable *string, startDate, endDate *string, page, pageSize int) ([]models.AuditLog, int, error) {
	offset := (page - 1) * pageSize
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if actorID != nil {
		whereClause += fmt.Sprintf(" AND a.actor_id = $%d", argIdx)
		args = append(args, *actorID)
		argIdx++
	}
	if action != nil && *action != "" {
		whereClause += fmt.Sprintf(" AND a.action = $%d", argIdx)
		args = append(args, *action)
		argIdx++
	}
	if targetTable != nil && *targetTable != "" {
		whereClause += fmt.Sprintf(" AND a.target_table = $%d", argIdx)
		args = append(args, *targetTable)
		argIdx++
	}
	if startDate != nil && *startDate != "" {
		whereClause += fmt.Sprintf(" AND a.created_at >= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil && *endDate != "" {
		whereClause += fmt.Sprintf(" AND a.created_at <= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, *endDate)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM audit_logs a %s`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.actor_id, u.username as actor_name, a.action, a.target_table, a.target_id, a.metadata, a.created_at
		FROM audit_logs a
		JOIN users u ON a.actor_id = u.id
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorName, &l.Action, &l.TargetTable, &l.TargetID, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
