package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// NotificationWorker memproses task notification:send dari antrian Asynq.
type NotificationWorker struct {
	notifications repository.NotificationRepository
	students      repository.StudentRepository
	users         repository.UserRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	notifications repository.NotificationRepository,
	students repository.StudentRepository,
	users repository.UserRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		notifications: notifications,
		students:      students,
		users:         users,
		tx:            tx,
		mailer:        mailer,
		logger:        logger,
	}
}

// HandleSend menangani satu task notification:send.
//
// Alur:
//  1. Unmarshal job, cari student → user penerima, bangun pesan.
//  2. Dalam SATU transaksi (WithTxOnce): Claim (idempotency) + mailer.Send.
//     Bila send gagal, transaksi rollback sehingga Claim ikut batal dan
//     percobaan ulang bisa mengklaim lagi.
//  3. !claimed (sudah terkirim) → return nil (lewati).
//  4. ErrPermanentDelivery → bungkus dengan asynq.SkipRetry.
//     ErrTemporaryDelivery → return error biasa (Asynq retry otomatis).
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		// Payload rusak: tidak akan benar pada retry berikutnya → skip.
		return fmt.Errorf("%w: unmarshal job: %w", asynq.SkipRetry, err)
	}

	// Cari penerima: student → owner user.
	var p enrollmentPayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		return fmt.Errorf("%w: unmarshal event payload: %w", asynq.SkipRetry, err)
	}

	student, err := w.students.FindByID(ctx, p.StudentID)
	if err != nil {
		return fmt.Errorf("%w: student not found: %w", asynq.SkipRetry, helper.ErrPermanentDelivery)
	}
	if student.OwnerID == nil {
		return fmt.Errorf("%w: student has no owner: %w", asynq.SkipRetry, helper.ErrPermanentDelivery)
	}
	user, err := w.users.FindByID(ctx, *student.OwnerID)
	if err != nil {
		return fmt.Errorf("%w: owner user not found: %w", asynq.SkipRetry, helper.ErrPermanentDelivery)
	}

	msg, err := BuildMessage(job.EventType, job.Payload, user)
	if err != nil {
		return fmt.Errorf("%w: build message: %w", asynq.SkipRetry, err)
	}

	// Claim + Send dalam satu transaksi.
	claimed := false
	txErr := w.tx.WithTxOnce(ctx, func(tx pgx.Tx) error {
		repo := w.notifications.WithTx(tx)

		ok, err := repo.Claim(ctx, model.Notification{
			OutboxID: job.OutboxID,
			UserID:   user.ID,
			Channel:  "email",
			Subject:  msg.Subject,
			Body:     msg.Body,
		})
		if err != nil {
			return err
		}
		if !ok {
			// Sudah pernah terkirim (idempotency key bentrok) → lewati.
			claimed = false
			return nil
		}
		claimed = true

		// Kirim; bila gagal, error membuat transaksi rollback
		// sehingga Claim ikut batal dan retry bisa mengklaim ulang.
		return w.mailer.Send(user.Email, msg.Subject, msg.Body)
	})

	if txErr != nil {
		// Kirim gagal (atau klaim gagal).
		if errors.Is(txErr, helper.ErrPermanentDelivery) {
			return fmt.Errorf("%w: %w", asynq.SkipRetry, txErr)
		}
		// Temporary (atau error DB) → biar Asynq mencoba ulang.
		return txErr
	}

	if !claimed {
		w.logger.Info("notifikasi sudah terkirim sebelumnya, dilewati",
			slog.Int64("outbox_id", job.OutboxID))
		return nil
	}

	w.logger.Info("notifikasi terkirim",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event_type", job.EventType),
		slog.Int("user_id", user.ID))
	return nil
}
