package helper

import "testing"

func TestHashPassword(t *testing.T) {
	password := "rahasia123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	// Hash harus berbeda dengan password asli
	if hash == password {
		t.Error("hash tidak boleh sama dengan password asli")
	}

	// Hash harus diawali $2a$12$ (bcrypt cost 12)
	if len(hash) < 7 || hash[:7] != "$2a$12$" {
		t.Errorf("hash harus diawali $2a$12$, dapat: %s", hash[:7])
	}

	// Verifikasi password benar
	if !VerifyPassword(hash, password) {
		t.Error("password yang benar harus lolos verifikasi")
	}

	// Verifikasi password salah
	if VerifyPassword(hash, "salah") {
		t.Error("password yang salah harus gagal verifikasi")
	}
}
