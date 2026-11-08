package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/helper"
)

// startWorker menyiapkan mailer, repository, relay outbox, dan worker Asynq.
// Mengembalikan fungsi shutdown untuk graceful stop (berhenti menerima task
// baru, selesaikan yang sedang berjalan, hentikan relay).
func startWorker(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) (func(context.Context) error, error) {
	// Mailer berbasis log (delay & failEvery dari env).
	mailerDelay := time.Duration(config.GetEnvInt("MAILER_DELAY_MS", 150)) * time.Millisecond
	mailerFailEvery := config.GetEnvInt("MAILER_FAIL_EVERY", 5)
	mailer := helper.NewLogMailer(mailerDelay, mailerFailEvery, "logs/mail.log")

	// Repository & TxManager.
	txManager := repository.NewTxManager(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	notifRepo := repository.NewNotificationRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)

	// Asynq client & worker server.
	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	srv := config.NewWorkerServer(logger)

	// Worker notifikasi.
	worker := service.NewNotificationWorker(notifRepo, studentRepo, userRepo, txManager, mailer, logger)

	// Relay outbox.
	relayInterval := time.Duration(config.GetEnvInt("RELAY_INTERVAL_MS", 1000)) * time.Millisecond
	relayBatch := config.GetEnvInt("RELAY_BATCH", 50)
	relay := service.NewOutboxRelay(outboxRepo, txManager, client, relayInterval, relayBatch, logger)

	//_MUX: daftarkan handler per task type.
	mux := asynq.NewServeMux()
	mux.HandleFunc(model.TaskNotificationSend, worker.HandleSend)

	// Jalankan worker server (blocking di goroutine).
	go func() {
		logger.Info("asynq worker starting",
			slog.Int("concurrency", config.GetEnvInt("WORKER_CONCURRENCY", 10)))
		if err := srv.Start(mux); err != nil {
			logger.Error("asynq worker stopped", slog.String("error", err.Error()))
		}
	}()

	// Jalankan relay outbox di background.
	relayCtx, cancelRelay := context.WithCancel(ctx)
	go relay.Run(relayCtx)

	// Fungsi graceful shutdown.
	shutdown := func(shutdownCtx context.Context) error {
		logger.Info("shutting down worker...")
		cancelRelay()   // hentikan relay lebih dulu
		srv.Stop()      // berhenti menerima task baru, selesaikan yang berjalan
		logger.Info("worker stopped gracefully")
		return nil
	}
	return shutdown, nil
}
