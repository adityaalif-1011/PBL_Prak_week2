package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

// ErrNotificationNotFound - notifikasi tidak ada.
var ErrNotificationNotFound = errors.New("notification not found")

// NotificationRepository menyimpan & membaca notifikasi terkirim.
type NotificationRepository interface {
	// Claim mencatat notifikasi baru dengan idempotency key outbox_id.
	// INSERT ... ON CONFLICT (outbox_id) DO NOTHING RETURNING id:
	// bila sudah pernah dikirim (konflik), tidak ada baris yang di-return
	// dan Claim mengembalikan false → worker boleh melewatkan (skip).
	Claim(ctx context.Context, n model.Notification) (bool, error)

	// FindByUser mengambil notifikasi milik user, terbaru di atas.
	FindByUser(ctx context.Context, userID int, limit int) ([]model.Notification, error)

	// Count menghitung seluruh notifikasi.
	Count(ctx context.Context) (int64, error)

	// WithTx mengembalikan salinan repository yang berjalan di dalam tx.
	WithTx(tx DBTX) NotificationRepository
}

type notificationPostgresRepository struct {
	db DBTX
}

func NewNotificationRepository(db DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: db}
}

// WithTx mengembalikan salinan repository yang berjalan di dalam tx.
func (r *notificationPostgresRepository) WithTx(tx DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: tx}
}

// Claim - idempotency via UNIQUE (outbox_id). true = berhasil diklaim.
func (r *notificationPostgresRepository) Claim(ctx context.Context, n model.Notification) (bool, error) {
	if n.Channel == "" {
		n.Channel = "email"
	}
	var id int64
	err := r.db.QueryRow(ctx,
		`INSERT INTO notifications (outbox_id, user_id, channel, subject, body)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (outbox_id) DO NOTHING
		 RETURNING id`,
		n.OutboxID, n.UserID, n.Channel, n.Subject, n.Body,
	).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Konflik: sudah pernah diklaim → bukan error, hanya tidak klaim.
			return false, nil
		}
		return false, fmt.Errorf("claim notification: %w", err)
	}
	return true, nil
}

// FindByUser - daftar notifikasi user (terbaru di atas via outbox_id DESC).
func (r *notificationPostgresRepository) FindByUser(ctx context.Context, userID, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, outbox_id, user_id, channel, subject, body, sent_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY outbox_id DESC
		 LIMIT $2`,
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("find notifications by user: %w", err)
	}
	defer rows.Close()

	list := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.OutboxID, &n.UserID, &n.Channel, &n.Subject, &n.Body, &n.SentAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// Count - total notifikasi terkirim.
func (r *notificationPostgresRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count notifications: %w", err)
	}
	return n, nil
}
