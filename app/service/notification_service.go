package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// NotificationService menyediakan pembacaan notifikasi & pemantauan antrian.
type NotificationService struct {
	notifications repository.NotificationRepository
	outbox        repository.OutboxRepository
	inspector     *asynq.Inspector
}

func NewNotificationService(
	notifications repository.NotificationRepository,
	outbox repository.OutboxRepository,
	inspector *asynq.Inspector,
) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		outbox:        outbox,
		inspector:     inspector,
	}
}

// Mine - GET /api/v1/notifications/mine (auth).
// Mengembalikan notifikasi milik user yang sedang login.
// @Summary Notifikasi saya
// @Description Mengambil notifikasi milik user yang sedang login (maksimal 50 terbaru).
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseEnvelope{data=[]model.Notification}
// @Failure 401 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /notifications/mine [get]
func (s *NotificationService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	list, err := s.notifications.FindByUser(ctx, current.UserID, 50)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.SuccessList(c, "notifications retrieved successfully", list, nil)
}

// Monitor - GET /api/v1/admin/queue (auth + permission queue:monitor).
// Menggabungkan stats outbox (Postgres), jumlah notifikasi, dan info queue
// (Redis). Bila Redis bermasalah, kembalikan error queue_error.
// @Summary Pantau antrian notifikasi
// @Description Menggabungkan statistik outbox (Postgres), jumlah notifikasi terkirim, dan info queue Asynq (Redis). Butuh permission queue:monitor.
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseEnvelope{data=model.QueueStats}
// @Failure 401 {object} helper.ErrorResponse
// @Failure 403 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Failure 503 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /admin/queue [get]
func (s *NotificationService) Monitor(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	outboxStats, err := s.outbox.Stats(ctx)
	if err != nil {
		return helper.InternalError(err)
	}

	notifCount, err := s.notifications.Count(ctx)
	if err != nil {
		return helper.InternalError(err)
	}

	// Info queue dari Redis; gagal → queue_error.
	var queues any
	qi, err := s.inspector.GetQueueInfo(model.QueueNotifications)
	if err != nil {
		return helper.NewError(fiber.StatusServiceUnavailable, "queue_error",
			"cannot reach redis queue: "+err.Error())
	}
	queues = qi

	return helper.Success(c, fiber.StatusOK, "queue stats retrieved", model.QueueStats{
		Outbox:        outboxStats,
		Notifications: notifCount,
		Queues:        queues,
	})
}
