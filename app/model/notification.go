package model

import (
	"encoding/json"
	"time"
)

// Konstanta event type yang dicatat ke outbox.
const (
	EventEnrollmentCreated = "enrollment.created"
	EventEnrollmentDropped = "enrollment.dropped"
	EventEnrollmentSwapped = "enrollment.swapped"
)

// Konstanta task type Asynq & nama queue.
const (
	TaskNotificationSend   = "notification:send"
	QueueNotifications     = "notifications"
)

// OutboxEvent - baris di tabel outbox (event yang menunggu direlay).
type OutboxEvent struct {
	ID          int64           `json:"id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
}

// NotificationJob - payload task Asynq (notification:send).
type NotificationJob struct {
	OutboxID  int64           `json:"outbox_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// Message - subjek & body email hasil BuildMessage.
type Message struct {
	Subject string
	Body    string
}

// Notification - baris di tabel notifications (sudah terkirim).
type Notification struct {
	ID        int64     `json:"id"`
	OutboxID  int64     `json:"outbox_id"`
	UserID    int       `json:"user_id"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}

// OutboxStats - ringkasan outbox untuk monitor.
type OutboxStats struct {
	Total       int64 `json:"total"`
	Unpublished int64 `json:"unpublished"`
	Published   int64 `json:"published"`
}

// QueueStats - ringkasan antrian untuk endpoint monitor.
type QueueStats struct {
	Outbox        OutboxStats `json:"outbox"`
	Notifications int64       `json:"notifications"`
	Queues        any         `json:"queues,omitempty"`
}
