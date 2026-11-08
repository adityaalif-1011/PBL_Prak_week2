//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/helper"
	"api-students/route"
)

const (
	apiJWTSecret = "0123456789abcdef0123456789abcdef0123456789abcdef"
	apiJWTIssuer = "api-students-integration"
	enrollTahun  = "2025/2026"
	batasSKS     = 24
	sksPerCourse = 3
	jumlahKursus = 10
	wantBerhasil = batasSKS / sksPerCourse // 8 pendaftaran yang muat
)

// newAPI membangun app Fiber lengkap (route + middleware asli) di atas
// PostgreSQL sungguhan lewat config.NewApp. MongoDB/Redis tidak dibutuhkan
// oleh endpoint enroll, jadi dependency itu tidak dipasang.
func newAPI(t *testing.T) (*fiber.App, *helper.JWTManager) {
	t.Helper()

	jwtManager := helper.NewJWTManager(apiJWTSecret, apiJWTIssuer, time.Hour)
	txManager := repository.NewTxManager(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	courseService := service.NewCourseService(courseRepo, txManager, outboxRepo)

	app := config.NewApp(pool, route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permissions:   helper.NewPermissionSet(nil),
		CourseService: courseService,
	})
	return app, jwtManager
}

// tokenFor membuat access token untuk user tertentu.
func tokenFor(t *testing.T, jwtManager *helper.JWTManager, user model.User) string {
	t.Helper()
	token, err := jwtManager.GenerateAccess(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

// enrollRequest mengirim POST /api/v1/enrollments (tanpa t.Fatal → aman
// dipakai dari goroutine).
func enrollRequest(app *fiber.App, token string, studentID, courseID int) (*http.Response, error) {
	body := fmt.Sprintf(`{"student_id":%d,"course_id":%d,"tahun_akademik":"%s"}`,
		studentID, courseID, enrollTahun)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	return app.Test(req, -1)
}

// enroll membungkus enrollRequest untuk pemanggil di goroutine test utama.
func enroll(t *testing.T, app *fiber.App, token string, studentID, courseID int) *http.Response {
	t.Helper()
	res, err := enrollRequest(app, token, studentID, courseID)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	return res
}

// countRows menghitung baris hasil query (biasanya COUNT(*)).
func countRows(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("countRows(%q): %v", query, err)
	}
	return n
}

// maxID mengambil id terbesar pada sebuah tabel (untuk mencari id yang pasti
// tidak ada).
func maxID(t *testing.T, table string) int {
	t.Helper()
	var id int
	if err := pool.QueryRow(context.Background(),
		fmt.Sprintf(`SELECT COALESCE(MAX(id), 0) FROM %s`, table)).Scan(&id); err != nil {
		t.Fatalf("maxID(%s): %v", table, err)
	}
	return id
}

// swapRequest mengirim POST /api/v1/enrollments/swap (tanpa t.Fatal → aman
// dipakai dari goroutine).
func swapRequest(app *fiber.App, token string, studentID, oldCourseID, newCourseID int) (*http.Response, error) {
	body := fmt.Sprintf(
		`{"student_id":%d,"old_course_id":%d,"new_course_id":%d,"tahun_akademik":"%s"}`,
		studentID, oldCourseID, newCourseID, enrollTahun)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap", strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	return app.Test(req, -1)
}

// dropRequest mengirim DELETE /api/v1/enrollments/:id.
func dropRequest(app *fiber.App, token string, enrollmentID int) (*http.Response, error) {
	req := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/v1/enrollments/%d", enrollmentID), nil)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	return app.Test(req, -1)
}

// readBody membaca seluruh body respons lalu menutupnya.
func readBody(t *testing.T, res *http.Response) []byte {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatalf("baca body: %v", err)
	}
	return body
}

// errorCode mengambil field code dari body error JSON (helper.ErrorResponse).
func errorCode(t *testing.T, res *http.Response) string {
	t.Helper()
	var e helper.ErrorResponse
	if err := json.Unmarshal(readBody(t, res), &e); err != nil {
		t.Fatalf("parse body error: %v", err)
	}
	return e.Code
}

// enrolledCount membaca enrolled_count sebuah course.
func enrolledCount(t *testing.T, courseID int) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT enrolled_count FROM courses WHERE id = $1`, courseID).Scan(&n); err != nil {
		t.Fatalf("enrolled_count(%d): %v", courseID, err)
	}
	return n
}

// enrollmentID mengambil id baris enrollment milik student pada course tertentu.
func enrollmentID(t *testing.T, studentID, courseID int) int {
	t.Helper()
	var id int
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID).Scan(&id); err != nil {
		t.Fatalf("enrollment id(%d,%d): %v", studentID, courseID, err)
	}
	return id
}

// countStudentEvents menghitung outbox bertipe event milik satu student.
func countStudentEvents(t *testing.T, studentID int, eventType string) int {
	t.Helper()
	return countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1 AND payload->>'student_id' = $2`,
		eventType, fmt.Sprintf("%d", studentID))
}

// TestEnrollEndpoint_WritesOutboxOnlyOnSuccess memastikan event outbox hanya
// ditulis ketika pendaftaran benar-benar sukses (transactional outbox):
// sukses → +1 baris; duplikat/kelas tak ada/student tak ada → tanpa baris baru.
func TestEnrollEndpoint_WritesOutboxOnlyOnSuccess(t *testing.T) {
	ctx := context.Background()
	userID, studentID := seedStudent(t)
	courseID := seedCourse(t, sksPerCourse, 10)

	app, jwtManager := newAPI(t)
	token := tokenFor(t, jwtManager, model.User{ID: userID, Username: fmt.Sprint(userID), Role: "user"})

	outboxBefore := countRows(t, `SELECT COUNT(*) FROM outbox`)

	// 1) Sukses → outbox bertambah tepat satu.
	res := enroll(t, app, token, studentID, courseID)
	if res.StatusCode != fiber.StatusCreated {
		t.Fatalf("sukses: status = %d, want %d", res.StatusCode, fiber.StatusCreated)
	}
	res.Body.Close()
	if got := countRows(t, `SELECT COUNT(*) FROM outbox`); got != outboxBefore+1 {
		t.Errorf("outbox setelah sukses = %d, want %d", got, outboxBefore+1)
	}

	// 2) Duplikat → 409, outbox tidak bertambah.
	dup := enroll(t, app, token, studentID, courseID)
	if dup.StatusCode != fiber.StatusConflict {
		t.Errorf("duplikat: status = %d, want %d", dup.StatusCode, fiber.StatusConflict)
	}
	dup.Body.Close()

	// 3) Kelas tidak ada → 404, outbox tidak bertambah.
	missingCourse := maxID(t, "courses") + 1000
	noCourse := enroll(t, app, token, studentID, missingCourse)
	if noCourse.StatusCode != fiber.StatusNotFound {
		t.Errorf("kelas tidak ada: status = %d, want %d", noCourse.StatusCode, fiber.StatusNotFound)
	}
	noCourse.Body.Close()

	// 4) Student tidak ada → 404, outbox tidak bertambah.
	missingStudent := maxID(t, "students") + 1000
	noStudent := enroll(t, app, token, missingStudent, courseID)
	if noStudent.StatusCode != fiber.StatusNotFound {
		t.Errorf("student tidak ada: status = %d, want %d", noStudent.StatusCode, fiber.StatusNotFound)
	}
	noStudent.Body.Close()

	if got := countRows(t, `SELECT COUNT(*) FROM outbox`); got != outboxBefore+1 {
		t.Errorf("outbox setelah semua penolakan = %d, want %d", got, outboxBefore+1)
	}

	// Hanya satu enrollment & beban SKS bertambah sekali.
	if got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1`, studentID); got != 1 {
		t.Errorf("jumlah enrollment = %d, want 1", got)
	}
	var totalSKS int
	if err := pool.QueryRow(ctx,
		`SELECT total_sks FROM students WHERE id = $1`, studentID).Scan(&totalSKS); err != nil {
		t.Fatalf("baca total_sks: %v", err)
	}
	if totalSKS != sksPerCourse {
		t.Errorf("total_sks = %d, want %d", totalSKS, sksPerCourse)
	}
}

// TestEnrollEndpoint_ConcurrentCreditLimit menjalankan 10 pendaftaran serentak
// untuk satu student dengan batas 24 SKS (3 SKS/kelas): harus TEPAT 8 yang
// berhasil. Mengunci baris student (SELECT ... FOR UPDATE) adalah kunci
// keberhasilan test ini — tanpanya batas SKS bisa terlampaui.
func TestEnrollEndpoint_ConcurrentCreditLimit(t *testing.T) {
	ctx := context.Background()
	userID, studentID := seedStudent(t)

	courseIDs := make([]int, jumlahKursus)
	for i := range courseIDs {
		courseIDs[i] = seedCourse(t, sksPerCourse, 10)
	}

	app, jwtManager := newAPI(t)
	token := tokenFor(t, jwtManager, model.User{ID: userID, Username: fmt.Sprint(userID), Role: "user"})

	// Pemanasan: bangun route tree sekali sebelum semua goroutine berlari.
	warm := httptest.NewRequest(http.MethodGet, "/api/v1/courses", strings.NewReader(""))
	warm.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	if _, err := app.Test(warm, -1); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	statuses := make([]int, jumlahKursus)
	errs := make([]error, jumlahKursus)

	for i := 0; i < jumlahKursus; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			res, err := enrollRequest(app, token, studentID, courseIDs[i])
			if err != nil {
				errs[i] = err
				return
			}
			statuses[i] = res.StatusCode
			_, _ = io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}(i)
	}
	close(start)
	wg.Wait()

	berhasil, ditolak := 0, 0
	for i := 0; i < jumlahKursus; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d: %v", i, errs[i])
			continue
		}
		switch statuses[i] {
		case fiber.StatusCreated:
			berhasil++
		case fiber.StatusConflict:
			ditolak++
		default:
			t.Errorf("goroutine %d: status tak terduga %d", i, statuses[i])
		}
	}

	if berhasil != wantBerhasil {
		t.Errorf("berhasil = %d, want tepat %d (status: %v)", berhasil, wantBerhasil, statuses)
	}
	if ditolak != jumlahKursus-wantBerhasil {
		t.Errorf("ditolak = %d, want %d", ditolak, jumlahKursus-wantBerhasil)
	}

	// Beban SKS student harus tepat 24 dan enrollment tersimpan 8.
	var totalSKS int
	if err := pool.QueryRow(ctx,
		`SELECT total_sks FROM students WHERE id = $1`, studentID).Scan(&totalSKS); err != nil {
		t.Fatalf("baca total_sks: %v", err)
	}
	if totalSKS != wantBerhasil*sksPerCourse {
		t.Errorf("total_sks = %d, want %d", totalSKS, wantBerhasil*sksPerCourse)
	}
	if totalSKS > batasSKS {
		t.Errorf("total_sks %d melebihi batas %d", totalSKS, batasSKS)
	}

	// Guard enrolled_count juga harus menahan tepat 8.
	var enrolledSum int
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(enrolled_count), 0) FROM courses WHERE id = ANY($1)`,
		courseIDs).Scan(&enrolledSum); err != nil {
		t.Fatalf("jumlah enrolled: %v", err)
	}
	if enrolledSum != wantBerhasil {
		t.Errorf("total enrolled_count = %d, want %d", enrolledSum, wantBerhasil)
	}
}

// TestSwapEndpoint_TargetFull memastikan swap menuju kelas yang sudah penuh
// ditolak dengan 409 course_full DAN keadaan tidak berubah sama sekali:
// mahasiswa tetap terdaftar di kelas asal, enrolled_count kedua kelas tidak
// bergerak, serta tidak ada event enrollment.swapped di outbox.
func TestSwapEndpoint_TargetFull(t *testing.T) {
	userID, studentID := seedStudent(t)
	otherUserID, otherStudentID := seedStudent(t)
	oldCourseID := seedCourse(t, sksPerCourse, 10)
	newCourseID := seedCourse(t, sksPerCourse, 1) // kuota 1 → cepat penuh

	app, jwtManager := newAPI(t)
	token := tokenFor(t, jwtManager, model.User{ID: userID, Username: fmt.Sprint(userID), Role: "user"})
	otherToken := tokenFor(t, jwtManager, model.User{ID: otherUserID, Username: fmt.Sprint(otherUserID), Role: "user"})

	// Penuhkan kelas tujuan dengan mahasiswa lain (1 dari 1 kuota terpakai).
	filler := enroll(t, app, otherToken, otherStudentID, newCourseID)
	if filler.StatusCode != fiber.StatusCreated {
		t.Fatalf("mengisi kelas tujuan: status = %d, want %d", filler.StatusCode, fiber.StatusCreated)
	}
	filler.Body.Close()

	// Daftarkan mahasiswa uji ke kelas asal.
	joined := enroll(t, app, token, studentID, oldCourseID)
	if joined.StatusCode != fiber.StatusCreated {
		t.Fatalf("enroll kelas asal: status = %d, want %d", joined.StatusCode, fiber.StatusCreated)
	}
	joined.Body.Close()

	oldBefore := enrolledCount(t, oldCourseID)
	newBefore := enrolledCount(t, newCourseID)

	// Swap ke kelas penuh → harus 409 course_full.
	res, err := swapRequest(app, token, studentID, oldCourseID, newCourseID)
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if res.StatusCode != fiber.StatusConflict {
		t.Errorf("swap ke kelas penuh: status = %d, want %d", res.StatusCode, fiber.StatusConflict)
	}
	if code := errorCode(t, res); code != "course_full" {
		t.Errorf("swap ke kelas penuh: code = %q, want %q", code, "course_full")
	}

	// Mahasiswa TETAP terdaftar di kelas asal.
	if got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, oldCourseID); got != 1 {
		t.Errorf("baris enrollment di kelas asal = %d, want 1", got)
	}

	// enrolled_count kedua kelas tidak berubah.
	if got := enrolledCount(t, oldCourseID); got != oldBefore {
		t.Errorf("enrolled_count kelas asal = %d, want %d", got, oldBefore)
	}
	if got := enrolledCount(t, newCourseID); got != newBefore {
		t.Errorf("enrolled_count kelas tujuan = %d, want %d", got, newBefore)
	}

	// Tidak ada event enrollment.swapped yang tercatat.
	if got := countStudentEvents(t, studentID, model.EventEnrollmentSwapped); got != 0 {
		t.Errorf("event enrollment.swapped = %d, want 0", got)
	}
}

// TestDropEndpoint_Success memastikan DELETE /enrollments/:id bekerja penuh:
// 204, baris enrollment hilang, enrolled_count berkurang satu, dan outbox
// berisi tepat satu event enrollment.dropped (transactional outbox).
func TestDropEndpoint_Success(t *testing.T) {
	userID, studentID := seedStudent(t)
	courseID := seedCourse(t, sksPerCourse, 10)

	app, jwtManager := newAPI(t)
	token := tokenFor(t, jwtManager, model.User{ID: userID, Username: fmt.Sprint(userID), Role: "user"})

	// Daftar dulu supaya ada baris yang bisa di-drop.
	joined := enroll(t, app, token, studentID, courseID)
	if joined.StatusCode != fiber.StatusCreated {
		t.Fatalf("enroll: status = %d, want %d", joined.StatusCode, fiber.StatusCreated)
	}
	joined.Body.Close()

	enrolledBefore := enrolledCount(t, courseID)
	if enrolledBefore != 1 {
		t.Fatalf("enrolled_count awal = %d, want 1", enrolledBefore)
	}
	if got := countStudentEvents(t, studentID, model.EventEnrollmentDropped); got != 0 {
		t.Fatalf("event enrollment.dropped sebelum drop = %d, want 0", got)
	}
	id := enrollmentID(t, studentID, courseID)

	// DELETE → 204.
	res, err := dropRequest(app, token, id)
	if err != nil {
		t.Fatalf("drop: %v", err)
	}
	if res.StatusCode != fiber.StatusNoContent {
		t.Errorf("drop: status = %d, want %d", res.StatusCode, fiber.StatusNoContent)
	}
	res.Body.Close()

	// Baris enrollment hilang.
	if got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID); got != 0 {
		t.Errorf("baris enrollment tersisa = %d, want 0", got)
	}

	// enrolled_count berkurang satu.
	if got := enrolledCount(t, courseID); got != enrolledBefore-1 {
		t.Errorf("enrolled_count = %d, want %d", got, enrolledBefore-1)
	}

	// Outbox berisi tepat satu event enrollment.dropped.
	if got := countStudentEvents(t, studentID, model.EventEnrollmentDropped); got != 1 {
		t.Errorf("event enrollment.dropped = %d, want 1", got)
	}
}
