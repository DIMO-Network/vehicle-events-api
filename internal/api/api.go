package api

import (
	"context"
	"github.com/DIMO-Network/shared"
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/vehicle-events-api/internal/config"
	"github.com/DIMO-Network/vehicle-events-api/internal/controllers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// Run sets up the API routes and starts the HTTP server.
func Run(ctx context.Context, logger zerolog.Logger, store db.Store) {
	logger.Info().Msg("Starting Vehicle Events API...")

	settings, err := shared.LoadConfig[config.Settings]("settings.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load settings")
	}
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "*",
		//AllowCredentials: true,
	}))

	app.Use(func(c *fiber.Ctx) error {
		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Msg("Middleware reached")

		// Hardcoded developer license for now
		devLicense := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
		c.Locals("developer_license_address", devLicense)
		return c.Next()
	})

	// Swagger setup
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	app.Get("/swagger.json", func(c *fiber.Ctx) error {
		return c.SendFile("./docs/swagger.json")
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the Vehicle Events API!")
	})

	// Register Webhook routes
	webhookController := controllers.NewWebhookController(store, logger)

	logger.Info().Msg("Registering routes...")
	app.Post("/webhooks", webhookController.RegisterWebhook)
	app.Get("/webhooks", webhookController.ListWebhooks)
	app.Put("/webhooks/:id", webhookController.UpdateWebhook)
	app.Delete("/webhooks/:id", webhookController.DeleteWebhook)
	app.Get("/webhooks/signals", webhookController.GetSignalNames)

	// Endpoint to build a CEL (Common Expression Language) expression from user-defined conditions
	// More info: https://github.com/google/cel-spec
	app.Post("/build-cel", webhookController.BuildCEL)

	// Register Vehicle Subscription routes
	vehicleSubscriptionController := controllers.NewVehicleSubscriptionController(store, logger)

	// New RESTful Routes
	app.Post("/subscriptions/:vehicleTokenID/event/:eventID", vehicleSubscriptionController.AssignVehicleToWebhook)
	app.Delete("/subscriptions/:vehicleTokenID/event/:eventID", vehicleSubscriptionController.RemoveVehicleFromWebhook)

	// Catchall
	app.Use(func(c *fiber.Ctx) error {
		logger.Warn().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Msg("404 Not Found")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not Found"})

	})

	// Start the server
	logger.Info().Msgf("Starting HTTP server on :%s...", settings.Port)
	if err := app.Listen(":" + settings.Port); err != nil {
		logger.Fatal().Err(err).Msg("Server failed to start")
	}
}
