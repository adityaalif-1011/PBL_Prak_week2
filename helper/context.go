package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

const LocalsAuthUser = "authUser"

// CurrentUser - ambil user dari context
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}
