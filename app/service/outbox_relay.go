package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"api-students/app/model"
	"api-students/app/repository"
)

// OutboxRelay membaca event outbox yang belum dipublish lalu melemparnya
// ke antrian Asynq. Pola ini memastikan event tidak pernah hilang:
// ditulis dalam transaksi bisnis, direlay secara at-least-once.
type OutboxRelay struct {
	outbox    repository.OutboxRepository
	tx        *repository.TxManager
	client    *asynq.Client
	interval  time.Duration
	batchSize int
	logger    *slog.Logger
}

// NewOutboxRelay membuat relay. interval: jeda antar siklus; batch: jumlah
// event per siklus.
func NewOutboxRelay(
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
	client *asynq.Client,
	interval time.Duration,
	batchSize int,
	logger *slog.Logger,
) *OutboxRelay {
	if interval <= 0 {
		interval = time.Second
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	return &OutboxRelay{
		outbox:    outbox,
		tx:        tx,
		client:    client,
		interval:  interval,
		batchSize: batchSize,
		logger:    logger,
	}
}

// Run menjalankan relay loop sampai ctx dibatalkan.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("outbox relay started",
		slog.Duration("interval", r.interval),
		slog.Int("batch", r.batchSize))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("outbox relay stopped")
			return
		case <-ticker.C:
			if err := r.RelayOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				r.logger.Error("outbox relay cycle failed", slog.String("error", err.Error()))
			}
		}
	}
}

// RelayOnce menjalankan satu siklus relay:
// 1) FetchUnpublished di dalam WithTxOnce (FOR UPDATE SKIP LOCKED).
// 2) EnqueueContext ke Asynq per event.
// 3) MarkPublished HANYA untuk yang berhasil di-enqueue.
func (r *OutboxRelay) RelayOnce(ctx context.Context) error {
	var events []model.OutboxEvent

	// Transaksi singkat: kunci baris selama dibaca (SKIP LOCKED) agar relay
	// paralel tidak memproses baris yang sama. Penguncian dilepas saat commit.
	err := r.tx.WithTxOnce(ctx, func(tx pgx.Tx) error {
		fetcher := r.outbox.WithTx(tx)
		evs, err := fetcher.FetchUnpublished(ctx, r.batchSize)
		if err != nil {
			return err
		}
		events = evs
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetch unpublished: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	published := make([]int64, 0, len(events))
	for _, e := range events {
		if err := r.enqueue(ctx, e); err != nil {
			// Event ini gagal; jangan ditandai published agar dicoba lagi
			// pada siklus berikutnya.
			r.logger.Error("gagal enqueue outbox event",
				slog.Int64("outbox_id", e.ID),
				slog.String("event_type", e.EventType),
				slog.String("error", err.Error()))
			continue
		}
		published = append(published, e.ID)
	}

	// Tandai hanya yang berhasil di-enqueue.
	if len(published) > 0 {
		if err := r.outbox.MarkPublished(ctx, published); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
		r.logger.Info("outbox relay published events",
			slog.Int("count", len(published)))
	}
	return nil
}

// enqueue melempar satu event ke queue notifications dengan TaskID deterministik
// (outbox-{id}) sehingga pengiriman ulang tidak menduplikasi task.
func (r *OutboxRelay) enqueue(ctx context.Context, e model.OutboxEvent) error {
	job := model.NotificationJob{
		OutboxID:  e.ID,
		EventType: e.EventType,
		Payload:   e.Payload,
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	task := asynq.NewTask(model.TaskNotificationSend, payload)
	opts := []asynq.Option{
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.MaxRetry(5),
		asynq.Retention(24 * time.Hour),
		asynq.Queue(model.QueueNotifications),
	}

	if _, err := r.client.EnqueueContext(ctx, task, opts...); err != nil {
		// TaskID sama sudah ada di antrian → dianggap sukses (idempoten).
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return nil
		}
		return err
	}
	return nil
}
