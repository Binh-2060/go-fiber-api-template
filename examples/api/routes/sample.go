package routes

import (
	"github.com/Binh-2060/go-application-template/examples/api/controllers"
	"github.com/gofiber/fiber/v3"
)

func SetSampleRoute(router fiber.Router) {
	router.Get("/", controllers.GetSampleController)
}
