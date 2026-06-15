package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type ReportRepository interface {
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]model.Report, int, error)
	Resolve(ctx context.Context, id uuid.UUID, action string) (*model.Report, error)
}

type reportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ReportRepository {
	return &reportRepository{db: db}
}

const reportColumns = `id, content, reporter, type, severity, status, action, created_at`

func (r *reportRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]model.Report, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reports WHERE status = $1`, status).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx, `SELECT `+reportColumns+` FROM reports WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	reports := []model.Report{}
	for rows.Next() {
		var rp model.Report
		if err := rows.Scan(&rp.ID, &rp.Content, &rp.Reporter, &rp.Type, &rp.Severity, &rp.Status, &rp.Action, &rp.CreatedAt); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		reports = append(reports, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return reports, total, nil
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
