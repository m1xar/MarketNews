package repository

import (
	"context"
	"database/sql"

	"MarketNews/internal/db/queries"
)

type MaintenanceRepository struct {
	db *sql.DB
}

func NewMaintenanceRepository(db *sql.DB) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

func (r *MaintenanceRepository) TruncateAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, queries.TruncateAll)
	return err
}
