package routes

import "github.com/gofiber/fiber/v3"

func SetRoutes(router fiber.Router) {
	// sample route
	sampleRoute := router.Group("/sample")
	SetSampleRoute(sampleRoute)
}
