package service

import (
	"testing"

	"api-students/app/model"
)

func TestValidateRegister_Valid(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sari",
		Email:    "sari@example.com",
		Password: "rahasia123",
	}
	errs := ValidateRegister(req)
	if len(errs) != 0 {
		t.Errorf("harusnya valid, dapat error: %v", errs)
	}
}

func TestValidateRegister_InvalidUsername(t *testing.T) {
	req := model.RegisterRequest{
		Username: "ab", // kurang dari 3
		Email:    "sari@example.com",
		Password: "rahasia123",
	}
	errs := ValidateRegister(req)
	if _, ok := errs["username"]; !ok {
		t.Error("harusnya ada error username")
	}
}

func TestValidateRegister_WeakPassword(t *testing.T) {
	req := model.RegisterRequest{
		Username: "budi",
		Email:    "budi@example.com",
		Password: "password1", // password umum
	}
	errs := ValidateRegister(req)
	if _, ok := errs["password"]; !ok {
		t.Error("harusnya ada error password")
	}
}

func TestValidateRegister_InvalidEmail(t *testing.T) {
	req := model.RegisterRequest{
		Username: "budi",
		Email:    "bukan-email",
		Password: "rahasia123",
	}
	errs := ValidateRegister(req)
	if _, ok := errs["email"]; !ok {
		t.Error("harusnya ada error email")
	}
}
