package controllers

import (
	"github.com/Binh-2060/go-application-template/examples/crud/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/crud/services"
	"github.com/Binh-2060/go-application-template/internal/api/presenters"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/gofiber/fiber/v3"
)

// Handlers take fiber.Ctx by value — in v3 Ctx is an interface, not a struct
// pointer. They stay thin on purpose: validate input, call a service, shape the
// output through presenters. Anything else belongs a layer down.

/*
POST /users/newData — create a user.
*/
func CreateUser(c fiber.Ctx) error {
	var body requestbody.CreateUser
	// Binds and runs the go-playground tags in one step. Fiber's own
	// StructValidator hook is not configured on this app, so tags are enforced
	// only when input goes through these helpers.
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := services.CreateUser(c.Context(), body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(user))
}

/*
GET /users/getData?page=&per_page=&q= — list users, paginated.
*/
func ListUsers(c fiber.Ctx) error {
	var query requestbody.ListUsers
	if err := validators.ParseAndValidateQueryParam(c, &query); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	page, err := services.ListUsers(c.Context(), query)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccessListData(
		page.Users, page.CurrentPage, page.CurrentPageTotalItem, page.TotalPage,
	))
}

/*
GET /users/info/:id — fetch one user.
*/
func GetUser(c fiber.Ctx) error {
	id := c.Params("id")
	if err := validators.ValidateUuid(id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := services.GetUser(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(user))
}

/*
PUT /users/update/:id — replace the user's fields.
*/
func UpdateUser(c fiber.Ctx) error {
	id := c.Params("id")
	if err := validators.ValidateUuid(id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var body requestbody.UpdateUser
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := services.UpdateUser(c.Context(), id, body); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess("SUCCESS"))
}

/*
DELETE /users/:id — delete one user.
*/
func DeleteUser(c fiber.Ctx) error {
	id := c.Params("id")
	if err := validators.ValidateUuid(id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := services.DeleteUser(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(nil))
}
