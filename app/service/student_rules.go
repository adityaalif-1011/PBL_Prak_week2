package service

import (
	"strings"

	"api-students/app/model"
)

// ValidateCreate memeriksa request POST
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM is required"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "name is required"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade must be between 0 and 100"
	}

	return errs
}

// ValidateReplace memeriksa request PUT (semua field wajib)
func ValidateReplace(req model.UpdateStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM is required"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "name is required"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade must be between 0 and 100"
	}

	return errs
}

// ApplyPatch menerapkan perubahan PATCH ke data yang ada
func ApplyPatch(current model.Student, req model.PatchStudentRequest) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			errs["nim"] = "NIM cannot be empty"
		} else {
			current.NIM = *req.NIM
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errs["name"] = "name cannot be empty"
		} else {
			current.Name = *req.Name
		}
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			errs["grade"] = "grade must be between 0 and 100"
		} else {
			current.Grade = *req.Grade
		}
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
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
