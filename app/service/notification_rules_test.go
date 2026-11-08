package service

import (
	"encoding/json"
	"strings"
	"testing"

	"api-students/app/model"
)

// TestBuildMessage memastikan setiap event enrollment menghasilkan subjek &
// body yang benar, dan event tak dikenal menghasilkan error.
func TestBuildMessage(t *testing.T) {
	user := model.User{ID: 7, Username: "sari", Email: "sari@example.com", Role: "user"}

	created := json.RawMessage(`{
		"student_id": 7,
		"course_id": 101,
		"course_kode": "IF101",
		"course_nama": "Basis Data",
		"tahun_akademik": "2025/2026"
	}`)
	swapped := json.RawMessage(`{
		"student_id": 7,
		"course_id": 101,
		"course_kode": "IF101",
		"course_nama": "Basis Data",
		"tahun_akademik": "2025/2026",
		"old_course_id": 102,
		"old_course_kode": "IF102",
		"old_course_nama": "Struktur Data"
	}`)

	t.Run("created", func(t *testing.T) {
		msg, err := BuildMessage(model.EventEnrollmentCreated, created, user)
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if msg.Subject != "Pendaftaran Mata Kuliah Berhasil" {
			t.Errorf("subject = %q", msg.Subject)
		}
		for _, want := range []string{"sari", "Basis Data", "IF101", "2025/2026"} {
			if !strings.Contains(msg.Body, want) {
				t.Errorf("body tidak memuat %q:\n%s", want, msg.Body)
			}
		}
	})

	t.Run("dropped", func(t *testing.T) {
		msg, err := BuildMessage(model.EventEnrollmentDropped, created, user)
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if msg.Subject != "Penghapusan Mata Kuliah Berhasil" {
			t.Errorf("subject = %q", msg.Subject)
		}
		for _, want := range []string{"sari", "Basis Data", "menghapus"} {
			if !strings.Contains(msg.Body, want) {
				t.Errorf("body tidak memuat %q:\n%s", want, msg.Body)
			}
		}
	})

	t.Run("swapped", func(t *testing.T) {
		msg, err := BuildMessage(model.EventEnrollmentSwapped, swapped, user)
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if msg.Subject != "Pindah Kelas Mata Kuliah Berhasil" {
			t.Errorf("subject = %q", msg.Subject)
		}
		for _, want := range []string{"sari", "Struktur Data", "IF102", "Basis Data", "IF101"} {
			if !strings.Contains(msg.Body, want) {
				t.Errorf("body tidak memuat %q:\n%s", want, msg.Body)
			}
		}
	})

	t.Run("unknown", func(t *testing.T) {
		if _, err := BuildMessage("enrollment.unknown", created, user); err == nil {
			t.Fatal("event tak dikenal harus menghasilkan error")
		} else if !strings.Contains(err.Error(), "unknown event type") {
			t.Errorf("pesan error = %q, want memuat \"unknown event type\"", err.Error())
		}
	})

	t.Run("payload rusak", func(t *testing.T) {
		if _, err := BuildMessage(model.EventEnrollmentCreated, json.RawMessage(`{rusak`), user); err == nil {
			t.Fatal("payload bukan JSON harus menghasilkan error")
		}
	})
}
