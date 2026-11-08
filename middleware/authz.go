package middleware

import (
	"github.com/gofiber/fiber/v2"

	"api-students/helper"
)

// RequirePermission menolak request yang role-nya tidak punya permission.
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
		}

		if !perms.Can(user.Role, permission) {
			return helper.NewError(fiber.StatusForbidden, "forbidden",
				"role "+user.Role+" tidak memiliki hak "+permission)
		}

		return c.Next()
	}
}
