package main

type Student struct {
	ID       int    `json:"id"`
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

type CreateStudentRequest struct {
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

type UpdateStudentRequest struct {
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	Grade    int    `json:"grade"`
	IsActive bool   `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string `json:"nim,omitempty"`
	Name     *string `json:"name,omitempty"`
	Grade    *int    `json:"grade,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *MetaData   `json:"meta,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

type MetaData struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

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

var students = []Student{
	{ID: 1, NIM: "2023001", Name: "Marcus Gideon", Grade: 85, IsActive: true},
	{ID: 2, NIM: "2023002", Name: "Aditya Alif", Grade: 92, IsActive: true},
	{ID: 3, NIM: "2023003", Name: "Florist Kristiani", Grade: 78, IsActive: false},
	{ID: 4, NIM: "2023004", Name: "Kevin Sanjaya", Grade: 88, IsActive: true},
	{ID: 5, NIM: "2023005", Name: "Hariman Retriver", Grade: 65, IsActive: true},
}

var lastID = 5
