package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"nonsoemeka-backend/internal/models"
)

type AuditWriter interface {
	Create(ctx context.Context, tx pgx.Tx, log models.AuditLog) error
}

func LogAction(ctx context.Context, repo AuditWriter, tx pgx.Tx, actorID uuid.UUID, action, targetTable string, targetID *uuid.UUID, metadata interface{}) error {
	var metaJSON json.RawMessage
	if metadata != nil {
		bytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal audit metadata: %w", err)
		}
		metaJSON = json.RawMessage(bytes)
	}

	log := models.AuditLog{
		ActorID:     actorID,
		Action:      action,
		TargetTable: targetTable,
		TargetID:    targetID,
		Metadata:    metaJSON,
	}

	return repo.Create(ctx, tx, log)
}
