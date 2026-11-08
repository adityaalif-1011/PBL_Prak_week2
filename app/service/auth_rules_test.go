package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestValidateRegister_Valid(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sari",
		Email:    "sari@example.com",
		Password: "rahasia123",
	}
	errs := helper.ValidateStruct(req)
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
	errs := helper.ValidateStruct(req)
	if _, ok := errs["username"]; !ok {
		t.Error("harusnya ada error username")
	}
}

func TestValidateRegister_UsernameWithSpace(t *testing.T) {
	req := model.RegisterRequest{
		Username: "sa ri",
		Email:    "sari@example.com",
		Password: "rahasia123",
	}
	errs := helper.ValidateStruct(req)
	if _, ok := errs["username"]; !ok {
		t.Error("harusnya ada error username karena mengandung spasi")
	}
}

func TestValidateRegister_WeakPassword(t *testing.T) {
	req := model.RegisterRequest{
		Username: "budi",
		Email:    "budi@example.com",
		Password: "password1", // password umum
	}
	errs := helper.ValidateStruct(req)
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
	errs := helper.ValidateStruct(req)
	if _, ok := errs["email"]; !ok {
		t.Error("harusnya ada error email")
	}
}

func TestValidateCreateUser_Invalid(t *testing.T) {
	req := model.CreateUserRequest{
		Username: "adit santoso",
		Email:    "bukan-email",
		Password: "pendek",
		Role:     "superadmin",
	}
	errs := helper.ValidateStruct(req)
	for _, field := range []string{"username", "email", "password", "role"} {
		if _, ok := errs[field]; !ok {
			t.Errorf("harusnya ada error untuk field %q", field)
		}
	}
}

func TestValidatePatchUser_SkipNil(t *testing.T) {
	req := model.PatchUserRequest{}
	if errs := helper.ValidateStruct(req); len(errs) != 0 {
		t.Errorf("patch kosong harus valid, dapat: %v", errs)
	}

	name := "adit santoso"
	req.Username = &name
	errs := helper.ValidateStruct(req)
	if _, ok := errs["username"]; !ok {
		t.Error("harusnya ada error username")
	}
}
