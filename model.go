package main

// Student struct dengan field yang diminta
// ID, Name, Grade, IsActive (dari tugas pertemuan 1) + NIM sebagai penanda unik
type Student struct {
	ID       int    `json:"id"`
	NIM      string `json:"nim"` // Penanda unik
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

// Request struct untuk POST (create)
type CreateStudentRequest struct {
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

// Request struct untuk PUT (update semua field)
type UpdateStudentRequest struct {
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

// Request struct untuk PATCH (update sebagian)
type PatchStudentRequest struct {
	NIM      *string `json:"nim,omitempty"`
	Name     *string `json:"name,omitempty"`
	Grade    *int    `json:"grade,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// Response envelope (amplop respons) untuk semua endpoint
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *MetaData   `json:"meta,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// Meta data untuk pagination
type MetaData struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Query params untuk filtering dan pagination
type QueryParams struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	Search   string `query:"search"`
	Sort     string `query:"sort"`
	Order    string `query:"order"`
	IsActive *bool  `query:"is_active"`
	MinGrade int    `query:"min_grade"`
	MaxGrade int    `query:"max_grade"`
}

// Data dummy (in-memory database)
var students = []Student{
	{ID: 1, NIM: "2023001", Name: "Budi Santoso", Grade: 85, IsActive: true},
	{ID: 2, NIM: "2023002", Name: "Siti Rahayu", Grade: 92, IsActive: true},
	{ID: 3, NIM: "2023003", Name: "Ahmad Fauzi", Grade: 78, IsActive: false},
	{ID: 4, NIM: "2023004", Name: "Dewi Lestari", Grade: 88, IsActive: true},
	{ID: 5, NIM: "2023005", Name: "Rizky Pratama", Grade: 65, IsActive: true},
}

var lastID = 5 // Untuk auto-increment ID
