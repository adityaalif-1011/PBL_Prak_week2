package service

import "strings"

// Aturan murni untuk review course (Modul 12 - Testing).
// Fungsi di file ini tidak bergantung pada fiber/repository sehingga bisa
// diuji sebagai unit test.

// NormalizeTags menormalkan daftar tag:
//   - buang tag yang hanya berisi spasi/kosong,
//   - potong spasi di sekeliling (strings.TrimSpace),
//   - seragamkan ke huruf kecil,
//   - buang duplikat, mempertahankan urutan kemunculan pertama.
//
// Input nil atau kosong menghasilkan slice kosong (bukan nil).
func NormalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))

	for _, tag := range tags {
		clean := strings.ToLower(strings.TrimSpace(tag))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		normalized = append(normalized, clean)
	}
	return normalized
}
