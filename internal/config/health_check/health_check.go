package healthcheck

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

/*
SetAppHealthCheck serves GET /healthz.

Mounted on its own path, not with app.Use: the handler answers every GET it
sees without calling Next, so as global middleware it swallowed every GET route.
*/
func SetAppHealthCheck(app *fiber.App) {
	app.Get("/healthz", healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
	}))
}
