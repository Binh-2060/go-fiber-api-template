package requestid

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func SetRequestIdMiddleware(app *fiber.App) {
	app.Use(requestid.New())
}
