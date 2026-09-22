package routes

import (
	"github.com/Binh-2060/go-application-template/examples/login/controllers"
	"github.com/Binh-2060/go-application-template/examples/login/middlewares"
	"github.com/gofiber/fiber/v3"
)

/*
Mount the login endpoint, and a RequireAuth-protected /me, on router.

Mirrors internal/api/routes: routes.go groups by feature path and delegates
to a Set<Feature>Route function like this one. To wire the example into the
running app, add to internal/api/routes/routes.go — see
examples/login/README.md for the full snippet, including the services.Init()
call this feature needs before any request reaches it.

	loginRoutes := router.Group("/login")
	loginroutes.SetAuthRoute(loginRoutes)

/me lives under the same group only for this example's convenience — a real
feature would mount its protected group wherever it needs it, not necessarily
beside login itself.

Protected routes go in a group carrying RequireAuth rather than each naming
the middleware itself. Both work, but with a group, "requires auth" is a
property of the group: a route added later inherits it, where a per-route
list has to remember it and silently serves unauthenticated traffic if it
doesn't.
*/
func SetAuthRoute(router fiber.Router) {
	router.Post("/", controllers.Login)

	protected := router.Group("/", middlewares.RequireAuth)
	protected.Get("/me", controllers.Me)
}
