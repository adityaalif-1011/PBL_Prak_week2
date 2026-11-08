package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

var (
	ErrCourseNotFound       = errors.New("course not found")
	ErrEnrollmentNotFound   = errors.New("enrollment not found")
	ErrEnrollmentDuplicate  = errors.New("enrollment already exists")
	ErrCourseQuotaExhausted = errors.New("course quota exhausted")
	ErrSKSLimitExceeded     = errors.New("SKS limit exceeded")
	ErrVersionConflict      = errors.New("course version conflict")
	ErrStudentNotFound      = errors.New("student not found")
)

// ReviewGate adalah cek prasyarat pembuatan review: kelas harus ada dan
// student harus terdaftar sebagai pesertanya. Dipakai ReviewService; berbentuk
// interface agar bisa dipalsukan (fake) di component test.
type ReviewGate interface {
	// CourseExists mengembalikan ErrCourseNotFound bila kelas tidak ada.
	CourseExists(ctx context.Context, courseID int) error
	// IsEnrolled mengembalikan ErrEnrollmentNotFound bila student bukan peserta.
	IsEnrolled(ctx context.Context, studentID, courseID int) error
}

// CourseRepository menangani course & enrollment.
// Semua method menerima DBTX sehingga bisa dipakai di dalam transaksi.
type CourseRepository struct {
	db DBTX
}

func NewCourseRepository(db DBTX) *CourseRepository {
	return &CourseRepository{db: db}
}

// WithTx mengembalikan salinan repository yang query-nya berjalan di dalam tx.
func (r *CourseRepository) WithTx(tx DBTX) *CourseRepository {
	return &CourseRepository{db: tx}
}

// ============================================================
// COURSES
// ============================================================

// FindAll - daftar course dengan sisa_kuota = kuota - enrolled_count.
func (r *CourseRepository) FindAll(ctx context.Context) ([]model.Course, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kode, nama, sks, kuota, enrolled_count,
		        (kuota - enrolled_count) AS sisa_kuota, version, created_at
		 FROM courses ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("find courses: %w", err)
	}
	defer rows.Close()

	list := []model.Course{}
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota,
			&c.EnrolledCount, &c.SisaKuota, &c.Version, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan course: %w", err)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// FindByID - ambil satu course berdasarkan id.
func (r *CourseRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`SELECT id, kode, nama, sks, kuota, enrolled_count,
		        (kuota - enrolled_count) AS sisa_kuota, version, created_at
		 FROM courses WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota,
		&c.EnrolledCount, &c.SisaKuota, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrCourseNotFound
		}
		return model.Course{}, fmt.Errorf("find course: %w", err)
	}
	return c, nil
}

// FindMyCourses - daftar mata kuliah yang diikuti student milik sebuah user
// (owner_id) beserta ringkasan beban SKS-nya. Bila user belum punya student
// atau belum mengambil mata kuliah apa pun, field terkait dikembalikan kosong.
func (r *CourseRepository) FindMyCourses(ctx context.Context, ownerID int) (model.MyCourses, error) {
	mine := model.MyCourses{Courses: []model.Enrollment{}}

	if err := r.db.QueryRow(ctx,
		`SELECT total_sks, batas_sks
		 FROM students
		 WHERE owner_id = $1 AND deleted_at IS NULL
		 ORDER BY id
		 LIMIT 1`,
		ownerID,
	).Scan(&mine.TotalCredits, &mine.MaxCredits); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.MyCourses{}, fmt.Errorf("find my credits: %w", err)
		}
		// User belum punya student: biarkan default (0/0) lalu tetap coba daftar.
	}

	rows, err := r.db.Query(ctx,
		`SELECT e.id, e.student_id, e.course_id, e.tahun_akademik, e.status, e.created_at,
		        c.kode, c.nama, c.sks
		 FROM students s
		 JOIN enrollments e ON e.student_id = s.id
		 JOIN courses c     ON c.id = e.course_id
		 WHERE s.owner_id = $1 AND s.deleted_at IS NULL
		 ORDER BY e.created_at DESC, e.id DESC`,
		ownerID,
	)
	if err != nil {
		return model.MyCourses{}, fmt.Errorf("find my courses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e model.Enrollment
		if err := rows.Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik,
			&e.Status, &e.CreatedAt, &e.Kode, &e.Nama, &e.SKS); err != nil {
			return model.MyCourses{}, fmt.Errorf("scan my course: %w", err)
		}
		mine.Courses = append(mine.Courses, e)
	}
	return mine, rows.Err()
}

// CourseExists - cek keberadaan kelas (prasyarat review).
func (r *CourseRepository) CourseExists(ctx context.Context, courseID int) error {
	var id int
	if err := r.db.QueryRow(ctx,
		`SELECT id FROM courses WHERE id = $1`, courseID,
	).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCourseNotFound
		}
		return fmt.Errorf("course exists: %w", err)
	}
	return nil
}

// IsEnrolled - cek apakah student terdaftar sebagai peserta kelas tersebut
// (semester mana pun, prasyarat review).
func (r *CourseRepository) IsEnrolled(ctx context.Context, studentID, courseID int) error {
	var id int
	if err := r.db.QueryRow(ctx,
		`SELECT id FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEnrollmentNotFound
		}
		return fmt.Errorf("is enrolled: %w", err)
	}
	return nil
}

// Create - tambah course baru.
func (r *CourseRepository) Create(ctx context.Context, req model.CreateCourseRequest) (model.Course, error) {
	kuota := req.Kuota
	var c model.Course
	err := r.db.QueryRow(ctx,
		`INSERT INTO courses (kode, nama, sks, kuota)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, kode, nama, sks, kuota, enrolled_count, version, created_at`,
		req.Kode, req.Nama, req.SKS, kuota,
	).Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota, &c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		return model.Course{}, fmt.Errorf("create course: %w", err)
	}
	c.SisaKuota = c.Kuota - c.EnrolledCount
	return c, nil
}

// LockCourse - kunci satu course (SELECT ... FOR UPDATE).
// Mengembalikan sisa_kuota terkini agar service bisa memutuskan.
func (r *CourseRepository) LockCourse(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`SELECT id, kode, nama, sks, kuota, enrolled_count,
		        (kuota - enrolled_count), version, created_at
		 FROM courses WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota,
		&c.EnrolledCount, &c.SisaKuota, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrCourseNotFound
		}
		return model.Course{}, fmt.Errorf("lock course: %w", err)
	}
	return c, nil
}

// LockCourses - kunci banyak course dengan urutan id tetap (ORDER BY id FOR UPDATE)
// untuk mencegah deadlock saat dua transaksi mengunci course yang sama.
func (r *CourseRepository) LockCourses(ctx context.Context, ids []int) (map[int]model.Course, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kode, nama, sks, kuota, enrolled_count,
		        (kuota - enrolled_count), version, created_at
		 FROM courses WHERE id = ANY($1) ORDER BY id FOR UPDATE`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("lock courses: %w", err)
	}
	defer rows.Close()

	result := map[int]model.Course{}
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota,
			&c.EnrolledCount, &c.SisaKuota, &c.Version, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan course: %w", err)
		}
		result[c.ID] = c
	}
	return result, rows.Err()
}

// IncrementEnrolled - tambah enrolled_count HANYA bila masih ada kuota.
// Guard WHERE enrolled_count < kuota membuat race condition mustahil lewat.
func (r *CourseRepository) IncrementEnrolled(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count + 1
		 WHERE id = $1 AND enrolled_count < kuota`, id)
	if err != nil {
		return fmt.Errorf("increment enrolled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCourseQuotaExhausted
	}
	return nil
}

// DecrementEnrolled - kurangi enrolled_count (untuk drop/swap).
func (r *CourseRepository) DecrementEnrolled(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count - 1
		 WHERE id = $1 AND enrolled_count > 0`, id)
	if err != nil {
		return fmt.Errorf("decrement enrolled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCourseNotFound
	}
	return nil
}

// UpdateQuota - ubah kuota dengan optimistic locking: version harus cocok,
// lalu version dinaikkan (version = version + 1).
func (r *CourseRepository) UpdateQuota(ctx context.Context, id, kuota, version int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`UPDATE courses SET kuota = $2, version = version + 1
		 WHERE id = $1 AND version = $3
		 RETURNING id, kode, nama, sks, kuota, enrolled_count, version, created_at`,
		id, kuota, version,
	).Scan(&c.ID, &c.Kode, &c.Nama, &c.SKS, &c.Kuota,
		&c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Bedakan "tidak ada" vs "version conflict"
			var exists int
			if e := r.db.QueryRow(ctx, `SELECT 1 FROM courses WHERE id = $1`, id).Scan(&exists); e == nil {
				return model.Course{}, ErrVersionConflict
			}
			return model.Course{}, ErrCourseNotFound
		}
		return model.Course{}, fmt.Errorf("update quota: %w", err)
	}
	c.SisaKuota = c.Kuota - c.EnrolledCount
	return c, nil
}

// ============================================================
// ENROLLMENTS
// ============================================================

// FindEnrollment - cek apakah student sudah terdaftar di course/semester itu.
func (r *CourseRepository) FindEnrollment(ctx context.Context, studentID, courseID int, tahun string) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.db.QueryRow(ctx,
		`SELECT id, student_id, course_id, tahun_akademik, status, created_at
		 FROM enrollments
		 WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3`,
		studentID, courseID, tahun,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.Status, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrEnrollmentNotFound
		}
		return model.Enrollment{}, fmt.Errorf("find enrollment: %w", err)
	}
	return e, nil
}

// CreateEnrollment - daftar course.
func (r *CourseRepository) CreateEnrollment(ctx context.Context, studentID, courseID int, tahun string) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.db.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, course_id, tahun_akademik, status)
		 VALUES ($1, $2, $3, 'active')
		 RETURNING id, student_id, course_id, tahun_akademik, status, created_at`,
		studentID, courseID, tahun,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.Status, &e.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Enrollment{}, ErrEnrollmentDuplicate
		}
		return model.Enrollment{}, fmt.Errorf("create enrollment: %w", err)
	}
	return e, nil
}

// FindEnrollmentByID - ambil enrollment + info course (untuk response).
func (r *CourseRepository) FindEnrollmentByID(ctx context.Context, id int) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.db.QueryRow(ctx,
		`SELECT e.id, e.student_id, e.course_id, e.tahun_akademik, e.status, e.created_at,
		        c.kode, c.nama, c.sks
		 FROM enrollments e JOIN courses c ON c.id = e.course_id
		 WHERE e.id = $1`,
		id,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.Status, &e.CreatedAt,
		&e.Kode, &e.Nama, &e.SKS)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrEnrollmentNotFound
		}
		return model.Enrollment{}, fmt.Errorf("find enrollment by id: %w", err)
	}
	return e, nil
}

// DeleteEnrollment - hapus pendaftaran (dipakai drop & swap).
func (r *CourseRepository) DeleteEnrollment(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM enrollments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}

// DeleteEnrollmentByKeys - hapus berdasar kunci natural (untuk swap).
func (r *CourseRepository) DeleteEnrollmentByKeys(ctx context.Context, studentID, courseID int, tahun string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM enrollments
		 WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3`,
		studentID, courseID, tahun)
	if err != nil {
		return fmt.Errorf("delete enrollment by keys: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}

// ============================================================
// STUDENT LOCKING (untuk transaksi enroll/swap)
// ============================================================

// LockStudent - kunci baris student (SELECT ... FOR UPDATE) agar pengecekan
// batas SKS aman dari race condition antar transaksi.
func (r *CourseRepository) LockStudent(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.db.QueryRow(ctx,
		`SELECT id, nim, name, grade, is_active, total_sks, batas_sks, owner_id, created_at
		 FROM students WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
		id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive,
		&s.TotalSKS, &s.BatasSKS, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrStudentNotFound
		}
		return model.Student{}, fmt.Errorf("lock student: %w", err)
	}
	return s, nil
}

// UpdateStudentSKS - tambah/kurangi total_sks student (delta bisa negatif).
// Guard menjaga total_sks tidak pernah negatif.
func (r *CourseRepository) UpdateStudentSKS(ctx context.Context, studentID, delta int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE students SET total_sks = total_sks + $2
		 WHERE id = $1 AND total_sks + $2 >= 0`,
		studentID, delta)
	if err != nil {
		return fmt.Errorf("update student sks: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStudentNotFound
	}
	return nil
}
