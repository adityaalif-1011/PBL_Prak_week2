//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
	"api-students/app/repository"
)

const tahunAkademik = "2025/2026"

// TestIncrementEnrolled_StopsAtQuota memastikan guard WHERE enrolled_count < kuota
// menghentikan increment tepat di kuota (tidak pernah melebihi).
func TestIncrementEnrolled_StopsAtQuota(t *testing.T) {
	ctx := context.Background()
	courseID := seedCourse(t, 3, 2)
	repo := repository.NewCourseRepository(pool)

	for i := 1; i <= 2; i++ {
		if err := repo.IncrementEnrolled(ctx, courseID); err != nil {
			t.Fatalf("increment ke-%d: %v", i, err)
		}
	}

	if err := repo.IncrementEnrolled(ctx, courseID); !errors.Is(err, repository.ErrCourseQuotaExhausted) {
		t.Fatalf("increment melewati kuota: err = %v, want ErrCourseQuotaExhausted", err)
	}

	var enrolled int
	if err := pool.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, courseID,
	).Scan(&enrolled); err != nil {
		t.Fatalf("baca enrolled_count: %v", err)
	}
	if enrolled != 2 {
		t.Errorf("enrolled_count = %d, want 2 (kuota)", enrolled)
	}
}

// TestInsertEnrollment_Duplicate memastikan unique constraint
// (student_id, course_id, tahun_akademik) menolak pendaftaran ganda.
func TestInsertEnrollment_Duplicate(t *testing.T) {
	ctx := context.Background()
	_, studentID := seedStudent(t)
	courseID := seedCourse(t, 3, 10)
	repo := repository.NewCourseRepository(pool)

	if _, err := repo.CreateEnrollment(ctx, studentID, courseID, tahunAkademik); err != nil {
		t.Fatalf("insert pertama: %v", err)
	}

	_, err := repo.CreateEnrollment(ctx, studentID, courseID, tahunAkademik)
	if !errors.Is(err, repository.ErrEnrollmentDuplicate) {
		t.Fatalf("insert kedua: err = %v, want ErrEnrollmentDuplicate", err)
	}

	got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID)
	if got != 1 {
		t.Errorf("jumlah enrollment = %d, want 1", got)
	}
}

// TestWithTx_RollsBackEveryStep memastikan SEMUA langkah dalam transaksi ikut
// ter-rollback saat callback mengembalikan error — termasuk catatan event di
// outbox (transactional outbox), enrolled_count, dan total_sks student.
func TestWithTx_RollsBackEveryStep(t *testing.T) {
	ctx := context.Background()
	_, studentID := seedStudent(t)
	courseID := seedCourse(t, 3, 10)

	txManager := repository.NewTxManager(pool)
	courseRepo := repository.NewCourseRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	outboxBefore := countRows(t, `SELECT COUNT(*) FROM outbox`)

	err := txManager.WithTx(ctx, func(tx pgx.Tx) error {
		repo := courseRepo.WithTx(tx)

		if _, err := repo.CreateEnrollment(ctx, studentID, courseID, tahunAkademik); err != nil {
			return err
		}
		if err := repo.IncrementEnrolled(ctx, courseID); err != nil {
			return err
		}
		if err := repo.UpdateStudentSKS(ctx, studentID, 3); err != nil {
			return err
		}
		if _, err := outboxRepo.Insert(ctx, tx, model.EventEnrollmentCreated, map[string]any{
			"student_id": studentID,
			"course_id":  courseID,
		}); err != nil {
			return err
		}
		return errors.New("gagal disengaja")
	})
	if err == nil {
		t.Fatal("WithTx harus meneruskan error dari callback")
	}

	// 1) enrollment tidak boleh tersimpan
	if got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID); got != 0 {
		t.Errorf("enrollment tersimpan = %d, want 0", got)
	}

	// 2) enrolled_count kembali nol
	var enrolled int
	if err := pool.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, courseID).Scan(&enrolled); err != nil {
		t.Fatalf("baca enrolled_count: %v", err)
	}
	if enrolled != 0 {
		t.Errorf("enrolled_count = %d, want 0", enrolled)
	}

	// 3) total_sks student kembali nol
	var totalSKS int
	if err := pool.QueryRow(ctx,
		`SELECT total_sks FROM students WHERE id = $1`, studentID).Scan(&totalSKS); err != nil {
		t.Fatalf("baca total_sks: %v", err)
	}
	if totalSKS != 0 {
		t.Errorf("total_sks = %d, want 0", totalSKS)
	}

	// 4) outbox ikut rollback
	if got := countRows(t, `SELECT COUNT(*) FROM outbox`); got != outboxBefore {
		t.Errorf("outbox = %d, want %d (ikut rollback)", got, outboxBefore)
	}
}

// TestClaim_OnlyOnce memastikan idempotency claim notifikasi: klaim pertama
// berhasil, klaim ulang dengan outbox_id sama gagal tanpa error
// (INSERT ... ON CONFLICT (outbox_id) DO NOTHING).
func TestClaim_OnlyOnce(t *testing.T) {
	ctx := context.Background()
	userID, studentID := seedStudent(t)

	var outboxID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO outbox (event_type, payload)
		 VALUES ($1, $2)
		 RETURNING id`,
		model.EventEnrollmentCreated,
		fmt.Sprintf(`{"student_id":%d,"course_id":1}`, studentID),
	).Scan(&outboxID); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	t.Cleanup(func() {
		cctx := context.Background()
		if _, err := pool.Exec(cctx, `DELETE FROM outbox WHERE id = $1`, outboxID); err != nil {
			t.Errorf("cleanup outbox: %v", err)
		}
	})

	notifRepo := repository.NewNotificationRepository(pool)
	notif := model.Notification{
		OutboxID: outboxID,
		UserID:   userID,
		Channel:  "email",
		Subject:  "Pendaftaran Mata Kuliah Berhasil",
		Body:     "uji klaim",
	}

	claimed, err := notifRepo.Claim(ctx, notif)
	if err != nil {
		t.Fatalf("klaim pertama: %v", err)
	}
	if !claimed {
		t.Fatal("klaim pertama harus berhasil (true)")
	}

	claimed, err = notifRepo.Claim(ctx, notif)
	if err != nil {
		t.Fatalf("klaim kedua tidak boleh error: %v", err)
	}
	if claimed {
		t.Fatal("klaim kedua harus ditolak (false)")
	}

	if got := countRows(t, `SELECT COUNT(*) FROM notifications WHERE outbox_id = $1`, outboxID); got != 1 {
		t.Errorf("jumlah notifikasi = %d, want 1", got)
	}
}
