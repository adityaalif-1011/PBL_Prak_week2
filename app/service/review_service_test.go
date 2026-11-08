package service_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/service"
	"api-students/config"
	"api-students/helper"
	"api-students/route"
)

const (
	testJWTSecret = "0123456789abcdef0123456789abcdef0123456789abcdef"
	testJWTIssuer = "api-students-test"
)

// apiResponse adalah bentuk umum response API (sukses maupun error).
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
}

// testEnv adalah lingkungan component test: app Fiber lengkap dengan fakes.
type testEnv struct {
	app     *fiber.App
	reviews *fakeReviewRepo
	courses *fakeCourseRepo
	token   string
}

// newTestEnv membangun app lewat config.NewApp (route & middleware asli)
// dengan repository diganti fake — tanpa PostgreSQL, MongoDB, maupun Redis.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	jwtManager := helper.NewJWTManager(testJWTSecret, testJWTIssuer, time.Hour)
	reviews := newFakeReviewRepo()
	courses := newFakeCourseRepo()

	app := config.NewApp(nil, route.Dependencies{
		JWT:           jwtManager,
		Permissions:   helper.NewPermissionSet(nil),
		ReviewService: service.NewReviewService(reviews, courses),
	})

	token, err := jwtManager.GenerateAccess(model.User{
		ID:       1,
		Username: "sari",
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	return &testEnv{app: app, reviews: reviews, courses: courses, token: token}
}

// do menjalankan request lewat app.Test(req, -1) — -1 tanpa batas waktu.
func do(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return res
}

// newRequest membuat request JSON dengan (opsional) bearer token.
func newRequest(method, path, body, token string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	}
	return req
}

// decode membaca & decode body response ke out.
func decode(t *testing.T, res *http.Response, out any) {
	t.Helper()
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("baca body: %v", err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode body %q: %v", raw, err)
	}
}

func TestListReviews(t *testing.T) {
	env := newTestEnv(t)
	env.courses.addCourse(101)

	env.reviews.addReview(model.Review{CourseID: 101, StudentID: 7, Rating: 5, Comment: "materi jelas"})
	env.reviews.addReview(model.Review{CourseID: 101, StudentID: 8, Rating: 4, Comment: "lumayan"})
	// Review milik kelas lain tidak boleh ikut terbawa.
	env.reviews.addReview(model.Review{CourseID: 999, StudentID: 7, Rating: 1, Comment: "kelas lain"})

	res := do(t, env.app, newRequest(http.MethodGet, "/api/v1/courses/101/reviews", "", env.token))
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, fiber.StatusOK)
	}

	var body apiResponse
	decode(t, res, &body)
	if !body.Success {
		t.Fatalf("success = false, message=%q", body.Message)
	}

	var page model.ReviewPage
	if err := json.Unmarshal(body.Data, &page); err != nil {
		t.Fatalf("decode ReviewPage: %v", err)
	}
	if page.CourseID != 101 {
		t.Errorf("course_id = %d, want 101", page.CourseID)
	}
	if len(page.Reviews) != 2 {
		t.Errorf("jumlah review = %d, want 2", len(page.Reviews))
	}
	if page.Meta.Total != 2 {
		t.Errorf("meta.total = %d, want 2", page.Meta.Total)
	}
}

func TestCreateReview_Success(t *testing.T) {
	env := newTestEnv(t)
	env.courses.addCourse(101)
	env.courses.enroll(7, 101)

	body := `{"student_id":7,"rating":5,"comment":"materi jelas","tags":[" Go ","GO","java"]}`
	res := do(t, env.app, newRequest(http.MethodPost, "/api/v1/courses/101/reviews", body, env.token))
	if res.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", res.StatusCode, fiber.StatusCreated)
	}

	var resp apiResponse
	decode(t, res, &resp)
	if !resp.Success {
		t.Fatalf("success = false, message=%q", resp.Message)
	}

	var created model.Review
	if err := json.Unmarshal(resp.Data, &created); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if created.CourseID != 101 || created.StudentID != 7 || created.Rating != 5 {
		t.Errorf("review tidak sesuai: %+v", created)
	}
	// Tag dinormalkan: trim, huruf kecil, duplikat dibuang.
	wantTags := []string{"go", "java"}
	if len(created.Tags) != len(wantTags) {
		t.Fatalf("tags = %v, want %v", created.Tags, wantTags)
	}
	for i, tag := range wantTags {
		if created.Tags[i] != tag {
			t.Errorf("tags[%d] = %q, want %q", i, created.Tags[i], tag)
		}
	}
	if n := env.reviews.count(); n != 1 {
		t.Errorf("jumlah review tersimpan = %d, want 1", n)
	}
}

func TestCreateReview_Rejected(t *testing.T) {
	cases := []struct {
		name       string
		courseID   string
		body       string
		enrolled   bool // apakah student peserta kelas
		wantStatus int
		wantCode   string
	}{
		{
			name:       "bukan peserta",
			courseID:   "101",
			body:       `{"student_id":7,"rating":5,"comment":"materi jelas"}`,
			enrolled:   false,
			wantStatus: fiber.StatusForbidden,
			wantCode:   "not_enrolled",
		},
		{
			name:       "kelas tidak ada",
			courseID:   "424242",
			body:       `{"student_id":7,"rating":5,"comment":"materi jelas"}`,
			enrolled:   true,
			wantStatus: fiber.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "rating 9",
			courseID:   "101",
			body:       `{"student_id":7,"rating":9,"comment":"materi jelas"}`,
			enrolled:   true,
			wantStatus: fiber.StatusUnprocessableEntity,
			wantCode:   "validation_failed",
		},
		{
			name:       "body bukan JSON",
			courseID:   "101",
			body:       "bukan-json-benar",
			enrolled:   true,
			wantStatus: fiber.StatusBadRequest,
			wantCode:   "bad_request",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.courses.addCourse(101)
			if tc.enrolled {
				env.courses.enroll(7, 101)
			}

			path := "/api/v1/courses/" + tc.courseID + "/reviews"
			res := do(t, env.app, newRequest(http.MethodPost, path, tc.body, env.token))

			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tc.wantStatus)
			}

			var resp apiResponse
			decode(t, res, &resp)
			if resp.Success {
				t.Errorf("success = true, padahal harus ditolak")
			}
			if resp.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", resp.Code, tc.wantCode)
			}
			if n := env.reviews.count(); n != 0 {
				t.Errorf("review tersimpan = %d, want 0", n)
			}
		})
	}
}

func TestCreateReview_Twice(t *testing.T) {
	env := newTestEnv(t)
	env.courses.addCourse(101)
	env.courses.enroll(7, 101)

	body := `{"student_id":7,"rating":5,"comment":"materi jelas"}`

	first := do(t, env.app, newRequest(http.MethodPost, "/api/v1/courses/101/reviews", body, env.token))
	if first.StatusCode != fiber.StatusCreated {
		t.Fatalf("percobaan pertama: status = %d, want %d", first.StatusCode, fiber.StatusCreated)
	}
	first.Body.Close()

	second := do(t, env.app, newRequest(http.MethodPost, "/api/v1/courses/101/reviews", body, env.token))
	if second.StatusCode != fiber.StatusConflict {
		t.Fatalf("percobaan kedua: status = %d, want %d", second.StatusCode, fiber.StatusConflict)
	}

	var resp apiResponse
	decode(t, second, &resp)
	if resp.Success {
		t.Errorf("percobaan kedua seharusnya ditolak (unique index course_id+student_id)")
	}
	if n := env.reviews.count(); n != 1 {
		t.Errorf("jumlah review = %d, want 1", n)
	}
}
