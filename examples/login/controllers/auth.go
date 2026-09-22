package controllers

import (
	"errors"

	"github.com/Binh-2060/go-application-template/examples/login/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/login/services"
	"github.com/Binh-2060/go-application-template/internal/api/presenters"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/gofiber/fiber/v3"
)

/*
POST /login — authenticate and mint an RS256 access token.
*/
func Login(c fiber.Ctx) error {
	var body requestbody.Login
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	token, err := services.Login(c.Context(), body)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(token))
}
