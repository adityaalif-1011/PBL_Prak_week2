package service

// Aturan murni untuk course/enrollment (Modul 8 & 9).
// Semua fungsi di file ini bebas dependensi: tidak ada fiber, repository,
// maupun database, sehingga bisa diuji sebagai unit test biasa.

// WithinCreditLimit mengecek apakah penambahan SKS masih muat di batas maksimum.
// Total SKS yang sama dengan batas (current+add == maxSKS) masih diizinkan,
// sehingga batas atas ditulis secara eksklusif: current+add < maxSKS+1.
func WithinCreditLimit(currentSKS, addSKS, maxSKS int) bool {
	return currentSKS+addSKS < maxSKS+1
}

// CreditsAfterSwap menghitung total SKS setelah pindah kelas:
// kurangi SKS kelas lama, tambahkan SKS kelas baru.
func CreditsAfterSwap(currentSKS, oldSKS, newSKS int) int {
	return currentSKS - oldSKS + newSKS
}
