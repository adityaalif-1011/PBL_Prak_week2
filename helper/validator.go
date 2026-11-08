package helper

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate dibuat sekali saat package di-load (satu instance untuk seluruh aplikasi).
var validate = newValidator()

var (
	usernamePattern     = regexp.MustCompile(`^[A-Za-z0-9._]{3,32}$`)
	whitespaceRunes     = " \t\n\r\f\v"
	commonWeakPasswords = map[string]bool{
		"password1": true, "12345678": true, "qwerty123": true,
		"admin123": true, "password123": true,
	}
)

// newValidator menyiapkan validator global beserta custom rule.
func newValidator() *validator.Validate {
	v := validator.New()

	// Nama field di error mengikuti tag json, bukan nama struct.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})

	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), whitespaceRunes)
	})

	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return usernamePattern.MatchString(fl.Field().String())
	})

	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return isStrongPassword(fl.Field().String())
	})

	return v
}

// ValidateStruct memvalidasi struct dan mengembalikan map field -> pesan.
func ValidateStruct(s any) map[string]string {
	fields := map[string]string{}

	err := validate.Struct(s)
	if err == nil {
		return fields
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return fields
	}

	for _, fe := range validationErrs {
		fields[fe.Field()] = messageFor(fe)
	}
	return fields
}

// messageFor mengubah satu kegagalan validasi menjadi pesan yang mudah dibaca.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah, panjang 3-32 karakter"
	case "strongpassword":
		return "minimal 8 karakter dan harus memuat huruf dan angka"
	case "email":
		return "format email tidak valid"
	case "min":
		return fmt.Sprintf("minimal %s karakter", fe.Param())
	case "max":
		return fmt.Sprintf("maksimal %s karakter", fe.Param())
	case "gte":
		return fmt.Sprintf("minimal %s", fe.Param())
	case "lte":
		return fmt.Sprintf("maksimal %s", fe.Param())
	case "oneof":
		return "nilai tidak diizinkan"
	default:
		return "tidak valid"
	}
}

// isStrongPassword - panjang, campuran huruf & angka, bukan password umum.
func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return false
	}

	return !commonWeakPasswords[password]
}
