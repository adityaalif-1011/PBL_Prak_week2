CREATE TABLE achievements (
    id SERIAL PRIMARY KEY,
    student_id INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    champion_level VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index untuk mempercepat query
CREATE INDEX idx_achievements_student_id ON achievements(student_id);