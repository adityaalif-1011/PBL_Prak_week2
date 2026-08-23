package main

import (
	"sort"
	"strings"
)

// Fungsi untuk cek NIM unik
func isNIMUnique(nim string, excludeID int) bool {
	for _, s := range students {
		if s.NIM == nim && s.ID != excludeID {
			return false
		}
	}
	return true
}

// Fungsi untuk filter students berdasarkan query params
func filterStudents(params QueryParams) []Student {
	result := make([]Student, 0)

	for _, s := range students {
		// Filter by search (nama, case insensitive)
		if params.Search != "" {
			if !strings.Contains(strings.ToLower(s.Name), strings.ToLower(params.Search)) {
				continue
			}
		}

		// Filter by is_active
		if params.IsActive != nil {
			if s.IsActive != *params.IsActive {
				continue
			}
		}

		// Filter by grade range
		if params.MinGrade > 0 && s.Grade < params.MinGrade {
			continue
		}
		if params.MaxGrade > 0 && s.Grade > params.MaxGrade {
			continue
		}

		result = append(result, s)
	}

	// Sorting dengan whitelist field
	if params.Sort != "" {
		allowedSorts := map[string]bool{
			"id": true, "nim": true, "name": true,
			"grade": true, "is_active": true,
		}

		if allowedSorts[params.Sort] {
			sort.Slice(result, func(i, j int) bool {
				switch params.Sort {
				case "id":
					return result[i].ID < result[j].ID
				case "nim":
					return result[i].NIM < result[j].NIM
				case "name":
					return result[i].Name < result[j].Name
				case "grade":
					return result[i].Grade < result[j].Grade
				case "is_active":
					return result[i].IsActive && !result[j].IsActive
				}
				return false
			})

			// Reverse jika order desc
			if params.Order == "desc" {
				sort.Slice(result, func(i, j int) bool {
					switch params.Sort {
					case "id":
						return result[i].ID > result[j].ID
					case "nim":
						return result[i].NIM > result[j].NIM
					case "name":
						return result[i].Name > result[j].Name
					case "grade":
						return result[i].Grade > result[j].Grade
					case "is_active":
						return !result[i].IsActive && result[j].IsActive
					}
					return false
				})
			}
		}
	}

	return result
}

// Fungsi untuk pagination
func paginate(data []Student, page, limit int) ([]Student, MetaData) {
	total := len(data)
	totalPages := (total + limit - 1) / limit

	// Default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// Batas atas limit = 100
	// Alasan: mencegah overload server dan menjaga performa
	if limit > 100 {
		limit = 100
	}

	start := (page - 1) * limit
	end := start + limit

	if start > total {
		return []Student{}, MetaData{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}
	}

	if end > total {
		end = total
	}

	meta := MetaData{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return data[start:end], meta
}
