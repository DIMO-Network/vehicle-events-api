package controllers

import (
	"github.com/DIMO-Network/shared/db"
	"github.com/DIMO-Network/vehicle-events-api/internal/db/models"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/types"
	"net/http"
	"time"
)

type VehicleSubscriptionController struct {
	store  db.Store
	logger zerolog.Logger
}

func NewVehicleSubscriptionController(store db.Store, logger zerolog.Logger) *VehicleSubscriptionController {
	return &VehicleSubscriptionController{
		store:  store,
		logger: logger,
	}
}

type SubscriptionView struct {
	EventID        string    `json:"event_id"`
	VehicleTokenID string    `json:"vehicle_token_id"`
	CreatedAt      time.Time `json:"created_at"`
	Description    string    `json:"description"`
	//condition? todo?
}

// AssignVehicleToWebhook godoc
// @Summary      Assign a vehicle to a webhook
// @Description  Associates a vehicle with a specific event webhook, optionally using conditions.
// @Tags         Vehicle Subscriptions
// @Accept       json
// @Produce      json
// @Param        vehicleTokenID path string true "Vehicle Token ID"
// @Param        eventID path string true "Event ID"
// @Param        request body object true "Request payload"
// @Success      201 "Vehicle assigned to webhook successfully"
// @Failure      400 "Invalid request payload or vehicle token ID"
// @Failure      401 "Unauthorized"
// @Failure      500 "Internal server error"
// @Security     BearerAuth
// @Router       /subscriptions/{vehicleTokenID}/event/{eventID} [post]
func (v *VehicleSubscriptionController) AssignVehicleToWebhook(c *fiber.Ctx) error {
	vehicleTokenIDStr := c.Params("vehicleTokenID")
	eventID := c.Params("eventID")

	if vehicleTokenIDStr == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Vehicle token ID is required"})
	}

	vehicleTokenIDDecimal := types.Decimal{}
	if err := vehicleTokenIDDecimal.Scan(vehicleTokenIDStr); err != nil {
		v.logger.Error().Err(err).Msg("Invalid vehicle token ID format")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid vehicle token ID format"})
	}

	var payload struct {
	}
	if err := c.BodyParser(&payload); err != nil {
		v.logger.Error().Err(err).Msg("Invalid request payload")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	devLicense, ok := c.Locals("developer_license_address").([]byte)
	if !ok {
		v.logger.Error().Msg("Developer license not found in request context")
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	eventVehicle := &models.EventVehicle{
		VehicleTokenID:          vehicleTokenIDDecimal,
		EventID:                 eventID,
		DeveloperLicenseAddress: devLicense,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := eventVehicle.Insert(c.Context(), v.store.DBS().Writer, boil.Infer()); err != nil {
		v.logger.Error().Err(err).Msg("Failed to assign vehicle to webhook")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign vehicle to webhook"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Vehicle assigned to webhook successfully"})
}

// RemoveVehicleFromWebhook godoc
// @Summary      Remove a vehicle from a webhook
// @Description  Unlinks a vehicle from a specific event webhook.
// @Tags         Vehicle Subscriptions
// @Produce      json
// @Param        vehicleTokenID path string true "Vehicle Token ID"
// @Param        eventID path string true "Event ID"
// @Success      200 "Vehicle removed from webhook successfully"
// @Failure      400 "Invalid vehicle token ID"
// @Failure      401 "Unauthorized"
// @Failure      500 "Internal server error"
// @Security     BearerAuth
// @Router       /subscriptions/{vehicleTokenID}/event/{eventID} [delete]
func (v *VehicleSubscriptionController) RemoveVehicleFromWebhook(c *fiber.Ctx) error {
	vehicleTokenIDStr := c.Params("vehicleTokenID")
	eventID := c.Params("eventID")

	vehicleTokenID, err := decimal.NewFromString(vehicleTokenIDStr)
	if err != nil {
		v.logger.Error().Err(err).Msg("Invalid vehicle token ID")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid vehicle token ID"})
	}

	devLicense, ok := c.Locals("developer_license_address").([]byte)
	if !ok {
		v.logger.Error().Msg("Developer license not found in request context")
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	_, err = models.EventVehicles(
		qm.Where("vehicle_token_id = ? AND event_id = ? AND developer_license_address = ?", vehicleTokenID, eventID, devLicense),
	).DeleteAll(c.Context(), v.store.DBS().Writer)
	if err != nil {
		v.logger.Error().Err(err).Msg("Failed to remove vehicle from webhook")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove vehicle from webhook"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Vehicle removed from webhook successfully"})
}

// ListSubscriptions godoc
// @Summary      List subscriptions for a vehicle
// @Description  Retrieves all webhook subscriptions for a given vehicle.
// @Tags         Vehicle Subscriptions
// @Produce      json
// @Param        vehicleTokenID path string true "Vehicle Token ID"
// @Success      200  {array}  object  "List of subscriptions"
// @Failure      401  "Unauthorized"
// @Failure      500  "Internal server error"
// @Security     BearerAuth
// @Router       /subscriptions/{vehicleTokenID} [get]
func (v *VehicleSubscriptionController) ListSubscriptions(c *fiber.Ctx) error {
	// Extract the vehicle token ID from the path
	vehicleTokenIDStr := c.Params("vehicleTokenID")
	if vehicleTokenIDStr == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Vehicle token ID is required"})
	}

	vehicleTokenID := types.Decimal{}
	if err := vehicleTokenID.Scan(vehicleTokenIDStr); err != nil {
		v.logger.Error().Err(err).Msg("Invalid vehicle token ID format")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid vehicle token ID format"})
	}

	devLicense, ok := c.Locals("developer_license_address").([]byte)
	if !ok {
		v.logger.Error().Msg("Developer license not found in request context")
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	subscriptions, err := models.EventVehicles(
		qm.Where("vehicle_token_id = ? AND developer_license_address = ?", vehicleTokenID, devLicense),
		qm.Load(models.EventVehicleRels.Event), //eager load
	).All(c.Context(), v.store.DBS().Reader)
	if err != nil {
		v.logger.Error().Err(err).Msg("Failed to retrieve subscriptions")
		return c.Status(fiber.StatusOK).JSON([]models.EventVehicle{})
	}

	resp := make([]SubscriptionView, 0, len(subscriptions))
	for _, sub := range subscriptions {
		var desc string
		if sub.R != nil && sub.R.Event != nil {
			desc = sub.R.Event.Description.String
		}

		view := SubscriptionView{
			EventID:        sub.EventID,
			VehicleTokenID: sub.VehicleTokenID.String(),
			CreatedAt:      sub.CreatedAt,
			Description:    desc,
		}

		resp = append(resp, view)
	}

	return c.JSON(resp)
}
