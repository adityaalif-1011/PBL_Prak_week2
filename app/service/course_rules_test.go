package service

import "testing"

// TestWithinCreditLimit memastikan batas SKS dihitung benar, khususnya pada
// batas (tepat muat vs melebihi satu SKS).
func TestWithinCreditLimit(t *testing.T) {
	cases := []struct {
		name    string
		current int
		add     int
		max     int
		want    bool
	}{
		{"masih jauh di bawah batas", 0, 3, 24, true},
		{"separuh terpakai", 12, 6, 24, true},
		{"tepat di batas", 21, 3, 24, true},
		{"sisa satu lalu penuh", 23, 1, 24, true},
		{"melebihi satu", 22, 3, 24, false},
		{"sudah penuh lalu tambah", 24, 1, 24, false},
		{"tanpa penambahan saat penuh", 24, 0, 24, true},
		{"batas nol tanpa tambahan", 0, 0, 0, true},
		{"batas nol dengan tambahan", 0, 1, 0, false},
		{"melebihi banyak", 20, 6, 24, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WithinCreditLimit(tc.current, tc.add, tc.max)
			if got != tc.want {
				t.Errorf("WithinCreditLimit(%d, %d, %d) = %v, want %v",
					tc.current, tc.add, tc.max, got, tc.want)
			}
		})
	}
}

// TestCreditsAfterSwap memastikan total SKS setelah pindah kelas dihitung
// dengan mengurangi kelas lama dan menambah kelas baru.
func TestCreditsAfterSwap(t *testing.T) {
	cases := []struct {
		name    string
		current int
		oldSKS  int
		newSKS  int
		want    int
	}{
		{"kelas dengan sks sama", 12, 3, 3, 12},
		{"naik sks", 12, 2, 4, 14},
		{"turun sks", 12, 4, 2, 10},
		{"swap dari kelas 2 ke 4 saat 6 terpakai", 6, 2, 4, 8},
		{"swap dari kelas 4 ke 2 saat 8 terpakai", 8, 4, 2, 6},
		{"mahasiswa semester pertama", 3, 3, 3, 3},
		{"menambah penuh dari nol", 0, 0, 3, 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CreditsAfterSwap(tc.current, tc.oldSKS, tc.newSKS)
			if got != tc.want {
				t.Errorf("CreditsAfterSwap(%d, %d, %d) = %d, want %d",
					tc.current, tc.oldSKS, tc.newSKS, got, tc.want)
			}
		})
	}
}
