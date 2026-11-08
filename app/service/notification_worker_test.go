package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
)

// fakeStudents meniru repository.StudentRepository: hanya FindByID yang hidup.
type fakeStudents struct {
	repository.StudentRepository

	students map[int]model.Student
}

func (f *fakeStudents) FindByID(_ context.Context, id int) (model.Student, error) {
	if s, ok := f.students[id]; ok {
		return s, nil
	}
	return model.Student{}, repository.ErrNotFound
}

// fakeUsers meniru repository.UserRepository: hanya FindByID yang hidup.
type fakeUsers struct {
	repository.UserRepository

	users map[int]model.User
}

func (f *fakeUsers) FindByID(_ context.Context, id int) (model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return model.User{}, repository.ErrUserNotFound
}

// newTask membungkus job menjadi task Asynq bertipe notification:send.
func newTask(t *testing.T, job model.NotificationJob) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	return asynq.NewTask(model.TaskNotificationSend, payload)
}

// TestHandleSend_PermanentFailures memastikan kegagalan yang tidak akan
// membaik bila di-retry (payload rusak, penerima tidak ada, event tak dikenal)
// selalu membungkus asynq.SkipRetry sehingga Asynq berhenti mencoba.
func TestHandleSend_PermanentFailures(t *testing.T) {
	ownerID := 11

	students := &fakeStudents{students: map[int]model.Student{
		7: {ID: 7, NIM: "221111", Name: "Sari", OwnerID: &ownerID},
	}}
	users := &fakeUsers{users: map[int]model.User{
		11: {ID: 11, Username: "sari", Email: "sari@example.com", Role: "user", IsActive: true},
	}}
	logger := slog.New(slog.DiscardHandler)

	validPayload := json.RawMessage(`{
		"student_id": 7,
		"course_id": 101,
		"course_kode": "IF101",
		"course_nama": "Basis Data",
		"tahun_akademik": "2025/2026"
	}`)

	// Jalur permanent failure tidak pernah menyentuh outbox/mailer/tx,
	// sehingga dependency itu boleh nil.
	newWorker := func(usersRepo repository.UserRepository) *NotificationWorker {
		return NewNotificationWorker(nil, students, usersRepo, nil, nil, logger)
	}

	cases := []struct {
		name   string
		worker *NotificationWorker
		task   *asynq.Task
	}{
		{
			name:   "payload bukan JSON",
			worker: newWorker(users),
			task:   asynq.NewTask(model.TaskNotificationSend, []byte("bukan-json")),
		},
		{
			name:   "user tidak ada",
			worker: newWorker(&fakeUsers{users: map[int]model.User{}}),
			task: newTask(t, model.NotificationJob{
				OutboxID:  1,
				EventType: model.EventEnrollmentCreated,
				Payload:   validPayload,
			}),
		},
		{
			name:   "student tidak ada",
			worker: NewNotificationWorker(nil, &fakeStudents{students: map[int]model.Student{}}, users, nil, nil, logger),
			task: newTask(t, model.NotificationJob{
				OutboxID:  2,
				EventType: model.EventEnrollmentCreated,
				Payload:   validPayload,
			}),
		},
		{
			name:   "event tidak dikenal",
			worker: newWorker(users),
			task: newTask(t, model.NotificationJob{
				OutboxID:  3,
				EventType: "enrollment.unknown",
				Payload:   validPayload,
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.worker.HandleSend(context.Background(), tc.task)
			if err == nil {
				t.Fatal("HandleSend harus mengembalikan error")
			}
			if !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("error harus membungkus asynq.SkipRetry, dapat: %v", err)
			}
		})
	}
}
