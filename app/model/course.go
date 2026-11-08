package model

import "time"

// Course - mata kuliah.
type Course struct {
	ID            int       `json:"id"`
	Kode          string    `json:"kode"`
	Nama          string    `json:"nama"`
	SKS           int       `json:"sks"`
	Kuota         int       `json:"kuota"`
	EnrolledCount int       `json:"enrolled_count"`
	SisaKuota     int       `json:"sisa_kuota"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateCourseRequest - request membuat course.
type CreateCourseRequest struct {
	Kode  string `json:"kode"  validate:"required,nospace,max=20"`
	Nama  string `json:"nama"  validate:"required,max=100"`
	SKS   int    `json:"sks"   validate:"required,gte=1,lte=6"`
	Kuota int    `json:"kuota" validate:"required,gte=1"`
}

// UpdateQuotaRequest - request mengubah kuota (optimistic lock via version).
type UpdateQuotaRequest struct {
	Kuota   int `json:"kuota"    validate:"required,gte=0"`
	Version int `json:"version"  validate:"gte=0"`
}

// Enrollment - pendaftaran mata kuliah oleh student.
type Enrollment struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	TahunAkademik string    `json:"tahun_akademik"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`

	// Join (opsional, diisi saat response)
	Kode string `json:"kode,omitempty"`
	Nama string `json:"nama,omitempty"`
	SKS  int    `json:"sks,omitempty"`
}

// CreateEnrollmentRequest - request enroll.
type CreateEnrollmentRequest struct {
	StudentID     int    `json:"student_id"     validate:"required,gt=0"`
	CourseID      int    `json:"course_id"      validate:"required,gt=0"`
	TahunAkademik string `json:"tahun_akademik" validate:"required,len=9"`
}

// SwapEnrollmentRequest - request pindah kelas dalam satu transaksi.
type SwapEnrollmentRequest struct {
	StudentID     int    `json:"student_id"       validate:"required,gt=0"`
	OldCourseID   int    `json:"old_course_id"    validate:"required,gt=0"`
	NewCourseID   int    `json:"new_course_id"    validate:"required,gt=0"`
	TahunAkademik string `json:"tahun_akademik"   validate:"required,len=9"`
}

// MyCourses - response GET /courses/mine: daftar mata kuliah yang diikuti
// student milik user yang login, plus ringkasan beban SKS-nya.
type MyCourses struct {
	// Courses daftar enrollment (khusus student milik user login).
	Courses []Enrollment `json:"courses"`
	// TotalCredits total SKS yang sedang diambil (students.total_sks).
	TotalCredits int `json:"total_credits"`
	// MaxCredits batas maksimum SKS (students.batas_sks).
	MaxCredits int `json:"max_credits"`
}
