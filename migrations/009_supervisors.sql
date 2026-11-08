-- 009_supervisors.sql
-- Modul 9: pembimbing TA (supervisors & supervisions)

CREATE TABLE IF NOT EXISTS supervisors (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL,
    max_students INT          NOT NULL DEFAULT 10 CHECK (max_students > 0),
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- student_id UNIQUE: satu student hanya punya satu pembimbing pada satu waktu.
CREATE TABLE IF NOT EXISTS supervisions (
    id            SERIAL PRIMARY KEY,
    supervisor_id INT NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    student_id    INT NOT NULL UNIQUE REFERENCES students(id) ON DELETE CASCADE,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supervisions_supervisor_id ON supervisions (supervisor_id);
