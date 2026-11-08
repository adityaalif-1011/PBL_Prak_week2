package service

import (
	"encoding/json"
	"fmt"

	"api-students/app/model"
)

// enrollmentPayload mencerminkan payload yang ditulis CourseService ke outbox.
type enrollmentPayload struct {
	StudentID     int    `json:"student_id"`
	CourseID      int    `json:"course_id"`
	CourseKode    string `json:"course_kode"`
	CourseNama    string `json:"course_nama"`
	TahunAkademik string `json:"tahun_akademik"`
	OldCourseID   int    `json:"old_course_id,omitempty"`
	OldCourseKode string `json:"old_course_kode,omitempty"`
	OldCourseNama string `json:"old_course_nama,omitempty"`
}

// BuildMessage menerjemahkan event type + payload menjadi subjek & body email.
// user dipakai untuk personalisasi (username).
func BuildMessage(eventType string, payload json.RawMessage, user model.User) (model.Message, error) {
	var p enrollmentPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return model.Message{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	switch eventType {
	case model.EventEnrollmentCreated:
		return model.Message{
			Subject: "Pendaftaran Mata Kuliah Berhasil",
			Body: fmt.Sprintf(
				"Halo %s,\n\nAnda telah berhasil mendaftar mata kuliah %s (%s) untuk semester %s.\n\nSalam,\nSIAKAD",
				user.Username, p.CourseNama, p.CourseKode, p.TahunAkademik),
		}, nil

	case model.EventEnrollmentDropped:
		return model.Message{
			Subject: "Penghapusan Mata Kuliah Berhasil",
			Body: fmt.Sprintf(
				"Halo %s,\n\nAnda telah berhasil menghapus mata kuliah %s (%s) dari KRS semester %s.\n\nSalam,\nSIAKAD",
				user.Username, p.CourseNama, p.CourseKode, p.TahunAkademik),
		}, nil

	case model.EventEnrollmentSwapped:
		return model.Message{
			Subject: "Pindah Kelas Mata Kuliah Berhasil",
			Body: fmt.Sprintf(
				"Halo %s,\n\nAnda telah berhasil berpindah dari %s (%s) ke %s (%s) untuk semester %s.\n\nSalam,\nSIAKAD",
				user.Username, p.OldCourseNama, p.OldCourseKode, p.CourseNama, p.CourseKode, p.TahunAkademik),
		}, nil

	default:
		return model.Message{}, fmt.Errorf("unknown event type: %s", eventType)
	}
}
