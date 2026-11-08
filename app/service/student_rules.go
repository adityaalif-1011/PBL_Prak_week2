package service

import (
	"api-students/app/model"
)

// ApplyPatch menerapkan field PATCH yang dikirim ke data yang ada.
// Validasi sudah dilakukan lewat helper.ValidateStruct sebelum fungsi ini.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = *req.NIM
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Grade != nil {
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current
}

// IsEmptyPatch cek apakah PATCH kosong
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// CountTotalPages hitung total halaman
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}
