package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Binh-2060/go-application-template/internal/api/routes"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/Binh-2060/go-application-template/internal/config/compress"
	"github.com/Binh-2060/go-application-template/internal/config/cors"
	"github.com/Binh-2060/go-application-template/internal/config/dotenv"
	healthcheck "github.com/Binh-2060/go-application-template/internal/config/health_check"
	"github.com/Binh-2060/go-application-template/internal/config/helmet"
	"github.com/Binh-2060/go-application-template/internal/config/limiter"
	"github.com/Binh-2060/go-application-template/internal/config/logger"
	"github.com/Binh-2060/go-application-template/internal/config/requestid"
	"github.com/Binh-2060/go-application-template/pkg/db"
	"github.com/gofiber/fiber/v3"
)

func init() {
	mode := os.Getenv("GO_ENV")
	if mode == "" {
		dotenv.SetDotenv()
	}

	//logging MODE of app
	mode = os.Getenv("GO_ENV")
	log.Println("------ Running in '" + mode + "' mode... ------")
}

func main() {
	var apiName = os.Getenv("API_NAME")
	var apiVersion = os.Getenv("API_VERSION")
	var mode = os.Getenv("GO_ENV")
	var buildAt = os.Getenv("BUILD_DATE")
	var startRunAt = time.Now().Format("2006-01-02 15:04:05")

	myConfig := fiber.Config{
		AppName: apiName,
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			// Status code defaults to 500
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			//response error
			err = ctx.Status(code).JSON(fiber.Map{
				"timestamp": time.Now().Format("2006-01-02-15-04-05"),
				"status":    0,
				"items":     nil,
				"error":     err.Error(),
			})
			return err
		},
	}

	//fail fast connect database
	if err := db.Init(context.Background(), db.ConfigFromEnv()); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	app := fiber.New(myConfig)
	//CORS
	cors.SetCORSMiddleware(app)
	// request ID
	requestid.SetRequestIdMiddleware(app)
	//validators
	validators.Init()
	//compress
	compress.SetCompressMiddleware(app)
	//helmet
	helmet.SetHelmetMiddleware(app)
	//limiter
	limiter.SetAppLimiter(app)
	//healthCheck
	healthcheck.SetAppHealthCheck(app)
	//group api
	api := app.Group("/api/" + apiVersion)
	//logging
	// Must be registered before any route on this group: Fiber only applies
	// middleware to routes added after it, so anything mounted above this line
	// is silently excluded from the access log.
	logger.SetLoggerMiddlewareJSON(api)

	api.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"API_NAME":     apiName,
			"API_VERSION":  apiVersion,
			"MODE":         mode,
			"BUILD_AT":     buildAt,
			"START_RUN_AT": startRunAt,
		})
	})

	//set api routes
	routes.SetRoutes(api)

	// Run server in a separate goroutine so it doesn't block
	go func() {
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Panic(err)
		}
	}()

	// Create channel to signify a signal being sent
	c := make(chan os.Signal, 1)

	// When an interrupt or termination signal is sent, notify the channel
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c // This blocks the main thread until an interrupt is received
	log.Println("Gracefully shutting down...")
	_ = app.Shutdown()

	log.Println("Running cleanup tasks...")
	// Close the pool only after Shutdown has drained in-flight requests, so no
	// handler is left holding a connection from a closed pool.
	db.Close()
	// Your cleanup tasks go here ...

	log.Println("Fiber was successful shutdown.")
}
