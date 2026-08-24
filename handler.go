package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// GET /api/v1/students - Daftar semua siswa dengan paginasi
func GetStudents(c *fiber.Ctx) error {
	var params QueryParams

	// Parse query params dengan default values yang aman
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.Limit, _ = strconv.Atoi(c.Query("limit", "10"))
	params.Search = c.Query("search", "")
	params.Sort = c.Query("sort", "")
	params.Order = c.Query("order", "asc")

	// Parse min_grade
	if minGrade := c.Query("min_grade"); minGrade != "" {
		params.MinGrade, _ = strconv.Atoi(minGrade)
	}

	// Parse max_grade
	if maxGrade := c.Query("max_grade"); maxGrade != "" {
		params.MaxGrade, _ = strconv.Atoi(maxGrade)
	}

	// Parse is_active
	if isActive := c.Query("is_active"); isActive != "" {
		val := isActive == "true"
		params.IsActive = &val
	}

	// Filter dan paginate
	filtered := filterStudents(params)
	paginated, meta := paginate(filtered, params.Page, params.Limit)

	// Return response dengan envelope
	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Message: "Students retrieved successfully",
		Data:    paginated,
		Meta:    &meta,
	})
}

// GET /api/v1/students/:id - Ambil satu siswa berdasarkan ID
func GetStudent(c *fiber.Ctx) error {
	// Parse ID dari parameter
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		// ID bukan angka -> 400 Bad Request
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid ID format",
			Errors:  "ID must be a number",
		})
	}

	// Cari siswa dengan ID tersebut
	for _, s := range students {
		if s.ID == id {
			return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
				Success: true,
				Message: "Student found",
				Data:    s,
			})
		}
	}

	// ID tidak ditemukan -> 404 Not Found
	return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
		Success: false,
		Message: "Student not found",
	})
}

// POST /api/v1/students - Tambah siswa baru
func CreateStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest

	// Parse JSON body
	if err := c.BodyParser(&req); err != nil {
		// Body bukan JSON yang valid -> 400 Bad Request
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid JSON body",
			Errors:  err.Error(),
		})
	}

	// Validasi: NIM required
	if req.NIM == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"nim": "NIM is required"},
		})
	}

	// Validasi: Name required
	if req.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"name": "Name is required"},
		})
	}

	// Validasi: Grade antara 0-100
	if req.Grade < 0 || req.Grade > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"grade": "Grade must be between 0 and 100"},
		})
	}

	// Cek duplikat NIM -> 409 Conflict
	if !isNIMUnique(req.NIM, 0) {
		return c.Status(fiber.StatusConflict).JSON(ResponseEnvelope{
			Success: false,
			Message: "NIM already exists",
			Errors:  map[string]string{"nim": "Duplicate NIM"},
		})
	}

	// Buat siswa baru
	lastID++
	student := Student{
		ID:       lastID,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}
	students = append(students, student)

	// Set Location header (201 Created)
	c.Set("Location", "/api/v1/students/"+strconv.Itoa(student.ID))

	// Return 201 Created
	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Message: "Student created successfully",
		Data:    student,
	})
}

// PUT /api/v1/students/:id - Update seluruh data siswa
func UpdateStudent(c *fiber.Ctx) error {
	// Parse ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid ID format",
			Errors:  "ID must be a number",
		})
	}

	var req UpdateStudentRequest

	// Parse JSON body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid JSON body",
			Errors:  err.Error(),
		})
	}

	// Validasi: semua field wajib diisi (PUT)
	if req.NIM == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"nim": "NIM is required"},
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"name": "Name is required"},
		})
	}
	if req.Grade < 0 || req.Grade > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string]string{"grade": "Grade must be between 0 and 100"},
		})
	}

	// Cari dan update siswa
	found := false
	for i, s := range students {
		if s.ID == id {
			// Cek duplikat NIM (exclude current)
			if !isNIMUnique(req.NIM, id) {
				return c.Status(fiber.StatusConflict).JSON(ResponseEnvelope{
					Success: false,
					Message: "NIM already exists",
					Errors:  map[string]string{"nim": "Duplicate NIM"},
				})
			}

			// Update semua field
			students[i].NIM = req.NIM
			students[i].Name = req.Name
			students[i].Grade = req.Grade
			students[i].IsActive = req.IsActive
			found = true
			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
			Success: false,
			Message: "Student not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Message: "Student updated successfully",
		Data:    students,
	})
}

// PATCH /api/v1/students/:id - Update sebagian data siswa
func PatchStudent(c *fiber.Ctx) error {
	// Parse ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid ID format",
			Errors:  "ID must be a number",
		})
	}

	var req PatchStudentRequest

	// Parse JSON body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid JSON body",
			Errors:  err.Error(),
		})
	}

	// Cari siswa
	var studentIndex int
	found := false
	for i, s := range students {
		if s.ID == id {
			studentIndex = i
			found = true
			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
			Success: false,
			Message: "Student not found",
		})
	}

	// Update hanya field yang dikirim (PATCH)
	if req.NIM != nil {
		if !isNIMUnique(*req.NIM, id) {
			return c.Status(fiber.StatusConflict).JSON(ResponseEnvelope{
				Success: false,
				Message: "NIM already exists",
				Errors:  map[string]string{"nim": "Duplicate NIM"},
			})
		}
		students[studentIndex].NIM = *req.NIM
	}

	if req.Name != nil {
		students[studentIndex].Name = *req.Name
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
				Success: false,
				Message: "Validation failed",
				Errors:  map[string]string{"grade": "Grade must be between 0 and 100"},
			})
		}
		students[studentIndex].Grade = *req.Grade
	}

	if req.IsActive != nil {
		students[studentIndex].IsActive = *req.IsActive
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Message: "Student patched successfully",
		Data:    students[studentIndex],
	})
}

// DELETE /api/v1/students/:id - Hapus siswa
func DeleteStudent(c *fiber.Ctx) error {
	// Parse ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Message: "Invalid ID format",
			Errors:  "ID must be a number",
		})
	}

	// Cari dan hapus siswa
	found := false
	for i, s := range students {
		if s.ID == id {
			students = append(students[:i], students[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
			Success: false,
			Message: "Student not found",
		})
	}

	// 204 No Content (tanpa body)
	return c.Status(fiber.StatusNoContent).JSON(nil)
}
