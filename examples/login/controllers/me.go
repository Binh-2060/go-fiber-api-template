package controllers

import (
	"github.com/Binh-2060/go-application-template/examples/login/middlewares"
	"github.com/Binh-2060/go-application-template/internal/api/presenters"
	"github.com/gofiber/fiber/v3"
)

/*
GET /me — returns the caller's user ID, proving RequireAuth ran and verified
the bearer token before this handler saw the request.

The !ok branch is not dead code: it's what this handler does if it's ever
mounted without RequireAuth. Discarding that bool would turn a wiring mistake
into a 200 for an unauthenticated caller.
*/
func Me(c fiber.Ctx) error {
	userID, ok := middlewares.UserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthenticated")
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(fiber.Map{
		"user_id": userID,
	}))
}
