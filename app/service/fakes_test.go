package service_test

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"api-students/app/model"
	"api-students/app/repository"
)

// Compile-time assertion: fake harus memenuhi kontrak repository.
var (
	_ repository.ReviewRepository = (*fakeReviewRepo)(nil)
	_ repository.ReviewGate       = (*fakeCourseRepo)(nil)
)

// fakeReviewRepo menyimpan review pada slice dan meniru perilaku unique index
// (course_id, student_id): satu student hanya boleh mengulas satu kelas sekali.
// Interface di-embed (nil) sehingga method yang tidak dipakai langsung panic
// bila tidak sengaja dipanggil — memaksa test hanya menyentuh jalur yang diuji.
type fakeReviewRepo struct {
	repository.ReviewRepository

	mu      sync.Mutex
	reviews []model.Review
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{}
}

func (f *fakeReviewRepo) EnsureIndexes(context.Context) error { return nil }

// Create - meniru unique index (course_id, student_id).
func (f *fakeReviewRepo) Create(_ context.Context, req model.CreateReviewRequest) (model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, r := range f.reviews {
		if r.CourseID == req.CourseID && r.StudentID == req.StudentID {
			return model.Review{}, repository.ErrDuplicate
		}
	}

	review := model.Review{
		ID:        bson.NewObjectID(),
		CourseID:  req.CourseID,
		StudentID: req.StudentID,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Tags:      req.Tags,
		Replies:   []model.Reply{},
		HelpfulBy: []int{},
		CreatedAt: time.Now().UTC(),
	}
	f.reviews = append(f.reviews, review)
	return review, nil
}

// FindByCourse - mengembalikan review satu kelas, terbaru di atas
// (meniru sort created_at DESC di MongoDB).
func (f *fakeReviewRepo) FindByCourse(_ context.Context, courseID int) ([]model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	list := []model.Review{}
	for _, r := range f.reviews {
		if r.CourseID == courseID {
			list = append(list, r)
		}
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

func (f *fakeReviewRepo) Summary(_ context.Context, courseID int) (model.ReviewSummary, error) {
	return model.ReviewSummary{
		CourseID:   courseID,
		RatingDist: map[string]int{},
	}, nil
}

// addReview menyuntikkan review langsung (untuk menyiapkan data test).
func (f *fakeReviewRepo) addReview(review model.Review) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if review.ID.IsZero() {
		review.ID = bson.NewObjectID()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now().UTC()
	}
	f.reviews = append(f.reviews, review)
}

func (f *fakeReviewRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reviews)
}

// fakeCourseRepo meniru prasyarat review: map kelas yang ada + map kelas yang
// diikuti tiap student.
type fakeCourseRepo struct {
	repository.ReviewGate

	mu       sync.Mutex
	courses  map[int]model.Course
	enrolled map[int]map[int]bool // studentID → set courseID
}

func newFakeCourseRepo() *fakeCourseRepo {
	return &fakeCourseRepo{
		courses:  map[int]model.Course{},
		enrolled: map[int]map[int]bool{},
	}
}

// addCourse menambahkan kelas yang dianggap ada.
func (f *fakeCourseRepo) addCourse(id int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.courses[id] = model.Course{ID: id, Kode: "IF101", Nama: "Basis Data", SKS: 3, Kuota: 40}
}

// enroll menandai student sebagai peserta kelas.
func (f *fakeCourseRepo) enroll(studentID, courseID int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.enrolled[studentID] == nil {
		f.enrolled[studentID] = map[int]bool{}
	}
	f.enrolled[studentID][courseID] = true
}

func (f *fakeCourseRepo) CourseExists(_ context.Context, courseID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.courses[courseID]; !ok {
		return repository.ErrCourseNotFound
	}
	return nil
}

func (f *fakeCourseRepo) IsEnrolled(_ context.Context, studentID, courseID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.enrolled[studentID][courseID] {
		return repository.ErrEnrollmentNotFound
	}
	return nil
}
