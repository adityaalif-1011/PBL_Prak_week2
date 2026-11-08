package model

import "time"

type Student struct {
	ID        int        `json:"id"`
	NIM       string     `json:"nim"`
	Name      string     `json:"name"`
	Grade     int        `json:"grade"`
	IsActive  bool       `json:"is_active"`
	TotalSKS  int        `json:"total_sks"`
	BatasSKS  int        `json:"batas_sks"`
	OwnerID   *int       `json:"owner_id,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM      string `json:"nim" validate:"required,nospace"`
	Name     string `json:"name" validate:"required"`
	Grade    int    `json:"grade" validate:"gte=0,lte=100"`
	IsActive bool   `json:"is_active"`
}

type UpdateStudentRequest struct {
	NIM      string `json:"nim" validate:"required,nospace"`
	Name     string `json:"name" validate:"required"`
	Grade    int    `json:"grade" validate:"gte=0,lte=100"`
	IsActive bool   `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string `json:"nim,omitempty" validate:"omitempty,nospace"`
	Name     *string `json:"name,omitempty" validate:"omitempty"`
	Grade    *int    `json:"grade,omitempty" validate:"omitempty,gte=0,lte=100"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type ListQuery struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	Search   string `query:"search"`
	Sort     string `query:"sort"`
	Order    string `query:"order"`
	IsActive *bool  `query:"is_active"`
	MinGrade int    `query:"min_grade"`
	MaxGrade int    `query:"max_grade"`
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

// Response envelope
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

type MetaData struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
