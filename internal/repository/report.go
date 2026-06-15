package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type ReportRepository interface {
	ListByStatus(ctx context.Context, status string) ([]model.Report, error)
	Resolve(ctx context.Context, id uuid.UUID, action string) (*model.Report, error)
}

type reportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ReportRepository {
	return &reportRepository{db: db}
}

const reportColumns = `id, content, reporter, type, severity, status, action, created_at`

func (r *reportRepository) ListByStatus(ctx context.Context, status string) ([]model.Report, error) {
	rows, err := r.db.Query(ctx, `SELECT `+reportColumns+` FROM reports WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	reports := []model.Report{}
	for rows.Next() {
		var rp model.Report
		if err := rows.Scan(&rp.ID, &rp.Content, &rp.Reporter, &rp.Type, &rp.Severity, &rp.Status, &rp.Action, &rp.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		reports = append(reports, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return reports, nil
}

func (r *reportRepository) Resolve(ctx context.Context, id uuid.UUID, action string) (*model.Report, error) {
	const query = `
		UPDATE reports SET status = 'resolved', action = $2
		WHERE id = $1
		RETURNING ` + reportColumns

	var rp model.Report
	err := r.db.QueryRow(ctx, query, id, action).Scan(
		&rp.ID, &rp.Content, &rp.Reporter, &rp.Type, &rp.Severity, &rp.Status, &rp.Action, &rp.CreatedAt,
	)
	if err != nil {
		return nil, apperror.NotFound("report not found")
	}
	return &rp, nil
}
