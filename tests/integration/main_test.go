//go:build integration

// Package integration berisi integration test yang menyentuh PostgreSQL
// sungguhan. Jalankan dengan:
//
//	go test -tags=integration -count=1 -v ./tests/integration/
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"api-students/config"
	"api-students/database"
)

// pool koneksi PostgreSQL bersama seluruh test (dibuka di TestMain).
var pool *pgxpool.Pool

// seedSeq penomoran unik untuk data seed (mencegah tabrakan NIM/username).
var seedSeq atomic.Int64

func TestMain(m *testing.M) {
	code := run(m)
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}

func run(m *testing.M) int {
	// 1) Muat .env.test (folder test atau root repo — go test menjalankan test
	//    dengan working directory = folder paket).
	_ = godotenv.Load(filepath.Join("..", "..", ".env.test"))

	// 2) Penjaga keamanan: database harus berakhiran _test.
	dbName := config.GetEnv("DB_NAME", "")
	if !strings.HasSuffix(dbName, "_test") {
		fmt.Fprintf(os.Stderr,
			"DB_NAME=%q tidak berakhiran _test — integration test menolak berjalan.\n"+
				"Salin .env menjadi .env.test (DB_NAME=praktikum_backend_test) "+
				"lalu jalankan: createdb praktikum_backend_test\n", dbName)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 3) Koneksi pgxpool.
	var err error
	pool, err = database.NewPool(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal terhubung ke database test:", err)
		return 1
	}

	// 4) Terapkan seluruh migration (idempoten lewat ledger schema_migrations).
	if err := applyMigrations(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "gagal apply migrations:", err)
		return 1
	}

	return m.Run()
}

// repoRoot mencari root repo (folder berisi go.mod) dari working directory.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// applyMigrations menjalankan semua file migrations/*.sql satu kali saja.
// Ledger schema_migrations membuat test bisa dijalankan berulang; bila skema
// sudah ada tanpa ledger (mis. dibuat manual), file tidak dieksekusi ulang
// agar migration yang memakai DROP/CREATE tidak menghapus data.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	dir := filepath.Join(repoRoot(), "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("baca %s: %w", dir, err)
	}

	files := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("buat ledger schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("baca ledger: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan ledger: %w", err)
		}
		applied[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Skema sudah ada tetapi ledger kosong → jangan eksekusi ulang (001 DROP TABLE).
	if len(applied) == 0 {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT to_regclass('students') IS NOT NULL`).Scan(&exists); err != nil {
			return fmt.Errorf("cek skema awal: %w", err)
		}
		if exists {
			for _, name := range files {
				if _, err := pool.Exec(ctx,
					`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`,
					name); err != nil {
					return fmt.Errorf("tandai migration %s: %w", name, err)
				}
			}
			return nil
		}
	}

	for _, name := range files {
		if applied[name] {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("baca migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("jalankan migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`,
			name); err != nil {
			return fmt.Errorf("catat migration %s: %w", name, err)
		}
	}
	return nil
}

// seedStudent membuat satu user + student baru.
// t.Cleanup menghapus student (enrollment ikut cascade), user, dan baris
// outbox yang menyinggung student tersebut.
func seedStudent(t *testing.T) (userID, studentID int) {
	t.Helper()
	ctx := context.Background()

	// NIM maksimal 20 karakter; angka unik dari UnixNano + penomuran seed.
	nim := fmt.Sprintf("T%d", time.Now().UnixNano()+seedSeq.Add(1))
	if len(nim) > 20 {
		nim = nim[:20]
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role, is_active)
		 VALUES ($1, $2, $3, 'user', true)
		 RETURNING id`,
		"u"+nim, nim+"@test.local", "x-not-a-real-hash",
	).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id, batas_sks)
		 VALUES ($1, $2, 80, true, $3, 24)
		 RETURNING id`,
		nim, "Mahasiswa Uji "+nim, userID,
	).Scan(&studentID); err != nil {
		t.Fatalf("seed student: %v", err)
	}

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Outbox milik student ini (event payload memuat student_id) → notifikasi
		// ikut terhapus karena FK ON DELETE CASCADE.
		if _, err := pool.Exec(cctx,
			`DELETE FROM outbox WHERE payload->>'student_id' = $1`,
			fmt.Sprintf("%d", studentID)); err != nil {
			t.Errorf("cleanup outbox: %v", err)
		}
		// Enrollment/achievement/supervision ikut cascade dari student.
		if _, err := pool.Exec(cctx, `DELETE FROM students WHERE id = $1`, studentID); err != nil {
			t.Errorf("cleanup student: %v", err)
		}
		if _, err := pool.Exec(cctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleanup user: %v", err)
		}
	})

	return userID, studentID
}

// seedCourse membuat satu course baru; t.Cleanup menghapusnya
// (enrollment ikut cascade).
func seedCourse(t *testing.T, sks, kuota int) int {
	t.Helper()
	ctx := context.Background()

	kode := fmt.Sprintf("UJI%d", time.Now().UnixNano()%1_000_000_000+seedSeq.Add(1))
	var id int
	if err := pool.QueryRow(ctx,
		`INSERT INTO courses (kode, nama, sks, kuota)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		kode, "Mata Kuliah Uji", sks, kuota,
	).Scan(&id); err != nil {
		t.Fatalf("seed course: %v", err)
	}

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cctx, `DELETE FROM courses WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup course: %v", err)
		}
	})

	return id
}
