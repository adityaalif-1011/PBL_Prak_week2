package helper

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Unix(1700000000, 123456789).UTC()
	cur := EncodeCursor(at, 42)

	gotAt, gotID, err := DecodeCursor(cur)
	if err != nil {
		t.Fatalf("decode gagal: %v", err)
	}
	if !gotAt.Equal(at) || gotID != 42 {
		t.Errorf("got (%v, %d), want (%v, 42)", gotAt, gotID, at)
	}

	if _, _, err := DecodeCursor("bukan-cursor"); err == nil {
		t.Error("cursor rusak harus error")
	}
}

func TestNegotiate(t *testing.T) {
	cases := map[string]string{
		"":                                       FormatJSON,
		"text/csv":                               FormatCSV,
		"application/json":                       FormatJSON,
		"*/*":                                    FormatJSON,
		"text/html":                              FormatJSON,
		"application/json;q=0.2, text/csv;q=0.9": FormatCSV,
	}

	for accept, want := range cases {
		app := fiber.New()
		app.Get("/", func(c *fiber.Ctx) error {
			if got := Negotiate(c); got != want {
				t.Errorf("Accept %q: got %s, want %s", accept, got, want)
			}
			return nil
		})

		req, err := http.NewRequest(fiber.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("buat request gagal: %v", err)
		}
		req.Header.Set(fiber.HeaderAccept, accept)

		if _, err := app.Test(req); err != nil {
			t.Fatalf("test request gagal: %v", err)
		}
	}
}

func TestValidateStructUserRequests(t *testing.T) {
	errs := ValidateStruct(model.RegisterRequest{
		Username: "a b",
		Email:    "salah",
		Password: "123",
	})
	for _, field := range []string{"username", "email", "password"} {
		if _, ok := errs[field]; !ok {
			t.Errorf("field %q harus punya error, dapat %v", field, errs)
		}
	}

	ok := ValidateStruct(model.RegisterRequest{
		Username: "sari.dev",
		Email:    "sari@example.com",
		Password: "rahasia123",
	})
	if len(ok) != 0 {
		t.Errorf("harusnya valid, dapat %v", ok)
	}
}
