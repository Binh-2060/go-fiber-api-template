package routes

import (
	"github.com/Binh-2060/go-application-template/examples/crud/controllers"
	"github.com/gofiber/fiber/v3"
)

/*
Mount the user CRUD endpoints on router.

Mirrors internal/api/routes: routes.go groups by feature path and delegates to a
Set<Feature>Route function like this one. To wire the example into the running
app, add to internal/api/routes/routes.go:

	userRoutes := router.Group("/users")
	crudroutes.SetUserRoute(userRoutes)

GET /getData and GET /info/:id never collide: /info/:id has two segments.
Only DELETE uses a bare /:id.
*/
func SetUserRoute(router fiber.Router) {
	router.Post("/newData", controllers.CreateUser)
	router.Get("/getData", controllers.ListUsers)
	router.Get("/info/:id", controllers.GetUser)
	router.Put("/update/:id", controllers.UpdateUser)
	router.Delete("/:id", controllers.DeleteUser)
}
