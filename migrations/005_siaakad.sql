-- 005_siaakad.sql
-- Modul 8 (UTS): tabel courses & enrollments + kolom SIAKAD pada students

-- ============================================================
-- students: tambah kolom SIAKAD + soft delete
-- ============================================================
ALTER TABLE students ADD COLUMN IF NOT EXISTS total_sks INT NOT NULL DEFAULT 0;
ALTER TABLE students ADD COLUMN IF NOT EXISTS batas_sks INT NOT NULL DEFAULT 24;
ALTER TABLE students ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- ============================================================
-- courses
-- ============================================================
CREATE TABLE IF NOT EXISTS courses (
    id              SERIAL PRIMARY KEY,
    kode            VARCHAR(20)  NOT NULL UNIQUE,
    nama            VARCHAR(100) NOT NULL,
    sks             INT          NOT NULL CHECK (sks > 0),
    kuota           INT          NOT NULL CHECK (kuota > 0),
    enrolled_count  INT          NOT NULL DEFAULT 0 CHECK (enrolled_count >= 0),
    version         INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- ============================================================
-- enrollments
-- ============================================================
CREATE TABLE IF NOT EXISTS enrollments (
    id              SERIAL PRIMARY KEY,
    student_id      INT          NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    course_id       INT          NOT NULL REFERENCES courses(id)  ON DELETE CASCADE,
    tahun_akademik  VARCHAR(9)   NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_unique UNIQUE (student_id, course_id, tahun_akademik)
);

CREATE INDEX IF NOT EXISTS idx_enrollments_student_id ON enrollments (student_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_id  ON enrollments (course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_tahun      ON enrollments (tahun_akademik);
