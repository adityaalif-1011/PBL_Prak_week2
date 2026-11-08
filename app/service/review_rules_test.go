package service

import (
	"slices"
	"testing"
)

// TestNormalizeTags memastikan tag dinormalkan: input kosong/nil, tag kosong,
// duplikat, dan huruf besar semuanya ditangani.
func TestNormalizeTags(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"input nil", nil, []string{}},
		{"input kosong", []string{}, []string{}},
		{"hanya spasi dan string kosong", []string{"", "   ", "\t"}, []string{}},
		{"duplikat dibuang", []string{"golang", "golang", "golang"}, []string{"golang"}},
		{"huruf besar diseragamkan", []string{"GOLANG", "JAVA"}, []string{"golang", "java"}},
		{"duplikat berbeda kapitalisasi", []string{"Go", "GO", "go"}, []string{"go"}},
		{
			name: "trim + urutan pertama dipertahankan",
			in:   []string{"  Go  ", "JAVA", "  ", "go", "python"},
			want: []string{"go", "java", "python"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeTags(tc.in)
			if !slices.Equal(got, tc.want) {
				t.Errorf("NormalizeTags(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if got == nil {
				t.Error("NormalizeTags tidak boleh mengembalikan nil")
			}
		})
	}
}
