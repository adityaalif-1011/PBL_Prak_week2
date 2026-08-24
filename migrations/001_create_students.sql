-- 001_create_students.sql
-- Description: Initialize students table

DROP TABLE IF EXISTS students;

CREATE TABLE students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    grade INT NOT NULL CHECK (grade >= 0 AND grade <= 100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_students_name ON students (name);
CREATE INDEX idx_students_is_active ON students (is_active);
CREATE INDEX idx_students_grade ON students (grade);