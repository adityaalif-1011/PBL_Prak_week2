package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/hibiken/asynq"

	"api-students/app/model"
)

// RedisOpt membaca konfigurasi Redis dari environment untuk Asynq.
func RedisOpt() asynq.RedisClientOpt {
	db, err := strconv.Atoi(GetEnv("REDIS_DB", "0"))
	if err != nil {
		db = 0
	}
	return asynq.RedisClientOpt{
		Addr:     GetEnv("REDIS_ADDR", "localhost:6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		DB:       db,
	}
}

// NewWorkerServer membuat Asynq server dengan konfigurasi dari env:
// concurrency, prioritas queue, dan retry delay bertahap (2,4,6,8,10 detik).
func NewWorkerServer(logger *slog.Logger) *asynq.Server {
	concurrency := GetEnvInt("WORKER_CONCURRENCY", 10)
	if concurrency < 1 {
		concurrency = 1
	}

	return asynq.NewServer(RedisOpt(), asynq.Config{
		Concurrency: concurrency,
		// Hanya queue notifications yang diprioritaskan (prio 1).
		Queues: map[string]int{
			model.QueueNotifications: 1,
		},
		// Retry delay: 2, 4, 6, 8, 10 detik (lalu tetap 10 detik).
		RetryDelayFunc: func(n int, err error, t *asynq.Task) time.Duration {
			delays := []time.Duration{
				2 * time.Second,
				4 * time.Second,
				6 * time.Second,
				8 * time.Second,
				10 * time.Second,
			}
			if n-1 >= 0 && n-1 < len(delays) {
				return delays[n-1]
			}
			return 10 * time.Second
		},
		// Log internal Asynq diarahkan ke slog.
		Logger: &asynqSlog{logger: logger},
	})
}

// asynqSlog menerjemahkan interface asynq.Logger ke slog.
// Interface asynq.Logger memakai signature Debug(args ...interface{}) dsb.
type asynqSlog struct {
	logger *slog.Logger
}

func (l *asynqSlog) Debug(args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *asynqSlog) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *asynqSlog) Warn(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *asynqSlog) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *asynqSlog) Fatal(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
	os.Exit(1)
}

// NewInspector membuat inspector Asynq (dipakai endpoint monitor).
func NewInspector() *asynq.Inspector {
	return asynq.NewInspector(RedisOpt())
}
