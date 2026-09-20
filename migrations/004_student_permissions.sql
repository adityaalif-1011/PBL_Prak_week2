-- 004_student_permissions.sql
-- Permission untuk entity students + kolom owner_id

-- Tambah permission students
INSERT INTO permissions (name, description) VALUES
  ('student:list',       'Melihat daftar seluruh student'),
  ('student:read:any',   'Melihat data student mana pun'),
  ('student:create',     'Menambah data student'),
  ('student:update:any', 'Mengubah data student mana pun'),
  ('student:delete',     'Menghapus data student')
ON CONFLICT (name) DO NOTHING;

-- Pasangkan ke role
INSERT INTO role_permissions (role_name, permission_name) VALUES
  ('admin', 'student:list'),
  ('admin', 'student:read:any'),
  ('admin', 'student:create'),
  ('admin', 'student:update:any'),
  ('admin', 'student:delete'),
  ('staff', 'student:list'),
  ('staff', 'student:read:any'),
  ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- Tambah kolom owner_id
ALTER TABLE students
  ADD COLUMN IF NOT EXISTS owner_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS students_owner_id_idx ON students (owner_id);