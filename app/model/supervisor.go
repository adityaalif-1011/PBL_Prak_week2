package model

import "time"

// Supervisor - dosen pembimbing TA.
type Supervisor struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	MaxStudents int       `json:"max_students"`
	CreatedAt   time.Time `json:"created_at"`
}

// Supervision - relasi pembimbing <-> student.
type Supervision struct {
	ID           int       `json:"id"`
	SupervisorID int       `json:"supervisor_id"`
	StudentID    int       `json:"student_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateSupervisorRequest - request membuat pembimbing.
type CreateSupervisorRequest struct {
	Name        string `json:"name"         validate:"required,max=100"`
	MaxStudents int    `json:"max_students" validate:"omitempty,gte=1"`
}

// PatchSupervisorRequest - ubah nama/kapasitas pembimbing.
type PatchSupervisorRequest struct {
	Name        *string `json:"name,omitempty"         validate:"omitempty,max=100"`
	MaxStudents *int    `json:"max_students,omitempty" validate:"omitempty,gte=1"`
}

// AssignSupervisionRequest - menugaskan student ke pembimbing.
type AssignSupervisionRequest struct {
	StudentID int `json:"student_id" validate:"required,gt=0"`
}

// SwitchSupervisionRequest - memindahkan student ke pembimbing lain.
type SwitchSupervisionRequest struct {
	StudentID     int `json:"student_id"      validate:"required,gt=0"`
	SupervisorID  int `json:"supervisor_id"   validate:"required,gt=0"`
}
