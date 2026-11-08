package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

// OutboxRepository menyimpan & membaca event outbox.
type OutboxRepository interface {
	// Insert mencatat event di dalam transaksi bisnis yang sama (dipanggil
	// dengan tx hasil WithTx). Karena satu transaksi, event hanya ikut
	// tersimpan bila bisnisnya commit (transactional outbox).
	Insert(ctx context.Context, tx pgx.Tx, eventType string, payload any) (model.OutboxEvent, error)

	// FetchUnpublished mengambil batch event belum dipublish dengan
	// FOR UPDATE SKIP LOCKED agar banyak relay bisa jalan paralel tanpa
	// memproses baris yang sama.
	FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error)

	// MarkPublished menandai event sudah direlay.
	MarkPublished(ctx context.Context, ids []int64) error

	// Stats menghitung ringkasan outbox.
	Stats(ctx context.Context) (model.OutboxStats, error)

	// WithTx mengembalikan salinan repository yang berjalan di dalam tx.
	WithTx(tx DBTX) OutboxRepository
}

type outboxPostgresRepository struct {
	db DBTX
}

func NewOutboxRepository(db DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: db}
}

// WithTx mengembalikan salinan repository yang berjalan di dalam tx.
func (r *outboxPostgresRepository) WithTx(tx DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: tx}
}

// Insert - mencatat event ke outbox DALAM transaksi bisnis (tx dari pemanggil).
func (r *outboxPostgresRepository) Insert(ctx context.Context, tx pgx.Tx, eventType string, payload any) (model.OutboxEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return model.OutboxEvent{}, fmt.Errorf("marshal outbox payload: %w", err)
	}

	var e model.OutboxEvent
	err = tx.QueryRow(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, $2)
		 RETURNING id, event_type, payload, created_at`,
		eventType, raw,
	).Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt)
	if err != nil {
		return model.OutboxEvent{}, fmt.Errorf("insert outbox: %w", err)
	}
	return e, nil
}

// FetchUnpublished - ambil batch event belum dipublish (FOR UPDATE SKIP LOCKED).
func (r *outboxPostgresRepository) FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, event_type, payload, created_at
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY id
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished outbox: %w", err)
	}
	defer rows.Close()

	events := []model.OutboxEvent{}
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkPublished - tandai event sudah direlay.
func (r *outboxPostgresRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE outbox SET published_at = NOW() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

// Stats - hitung total/unpublished/published.
func (r *outboxPostgresRepository) Stats(ctx context.Context) (model.OutboxStats, error) {
	var s model.OutboxStats
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE published_at IS NULL),
		        COUNT(*) FILTER (WHERE published_at IS NOT NULL)
		 FROM outbox`,
	).Scan(&s.Total, &s.Unpublished, &s.Published)
	if err != nil {
		return model.OutboxStats{}, fmt.Errorf("outbox stats: %w", err)
	}
	return s, nil
}
