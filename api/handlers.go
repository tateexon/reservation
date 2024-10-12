package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/tateexon/reservation/db"
	"github.com/tateexon/reservation/schema"
	"github.com/tateexon/reservation/utils"
)

// Server implementation
type Server struct {
	DB *db.Database
}

// Ensure that Server implements ServerInterface
var _ schema.ServerInterface = (*Server)(nil)

//nolint:revive
func (s *Server) GetAppointments(c *gin.Context, params schema.GetAppointmentsParams) {
	// Parse query parameters
	providerID := params.ProviderID
	date := params.Date

	// Get available appointment slots from the database
	slots, err := s.DB.GetAvailableAppointments(&providerID, &date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch appointments"})
		return
	}

	c.JSON(http.StatusOK, slots)
}

const (
	PostAppointmentsUnavailableTimeSlot string = "Time slot not available"
	PostAppointmentsInvalidID           string = "Invalid availability id"
	PostAppointmentsInvalidStartTime    string = "Reservations must be made at least 24 hours in advance"
	PostAppointmentsFailedToReserve     string = "Failed to reserve appointment"
)

func (s *Server) PostAppointments(c *gin.Context) {
	var req schema.PostAppointmentsJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// get appointment
	startTime, err := s.DB.GetAppointmentStartTime(req.AvailabilityId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": PostAppointmentsInvalidID})
		return
	}

	// Business logic checks
	if !s.isReservationAtLeast24HoursInAdvance(&startTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": PostAppointmentsInvalidStartTime})
		return
	}

	// Reserve the appointment
	appointment, err := s.DB.ReserveAppointment(req.ClientId, req.ProviderId, &startTime)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrReserveAppointmentSlotNotAvailable):
			c.JSON(http.StatusConflict, gin.H{"error": PostAppointmentsUnavailableTimeSlot})
			return
		default:
			log.Println("appointment reservation failure: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": PostAppointmentsFailedToReserve})
		}

		return
	}

	c.JSON(http.StatusCreated, appointment)
}

func (s *Server) isReservationAtLeast24HoursInAdvance(startTime *time.Time) bool {
	return time.Until(*startTime) >= 24*time.Hour
}

func (s *Server) PostAppointmentsAppointmentIDConfirm(c *gin.Context, appointmentID openapi_types.UUID) {
	// Confirm the reservation
	err := s.DB.ConfirmAppointment(appointmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Appointment not found or may have expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm appointment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment confirmed"})
}

const (
	// success constants
	postProvidersAvailabilityAdded string = "Availability added"

	// failure constants
	postProvidersAvailabilityFailToAdd        string = "Failed to add availability"
	postProvidersAvailabilityInvalidBody      string = "Invalid request body"
	postProvidersAvailabilityInvalidTimeRange string = "End time must be later than start time and at least 15 minutes apart"
)

func (s *Server) PostProvidersAvailability(c *gin.Context) {
	var availability schema.Availability

	if err := c.ShouldBindJSON(&availability); err != nil {
		log.Println("error parsing body: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": postProvidersAvailabilityInvalidBody})
		return
	}

	startTime := roundUpToNearestInterval(*availability.StartTime)
	// add a microsecond to the end so the .Before will include it
	endTime := roundDownToNearestInterval(*availability.EndTime).Add(time.Microsecond)

	// Validate time range
	if !areAtLeastTheIntervalApart(startTime, endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": postProvidersAvailabilityInvalidTimeRange})
		return
	}

	// Split time range into 15-minute slots
	slots := utils.GenerateTimeSlots(startTime, endTime, db.GetAvailabilityInterval())

	// Save availability slots to the database
	err := s.DB.AddAvailability(*availability.ProviderId, slots)
	if err != nil {
		log.Println("error adding availability: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": postProvidersAvailabilityFailToAdd})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": postProvidersAvailabilityAdded})
}

// Round up to the nearest 15-minute interval
func roundUpToNearestInterval(t time.Time) time.Time {
	t = roundDownToNearestInterval(t)
	return t.Add(db.GetAvailabilityInterval())
}

func roundDownToNearestInterval(t time.Time) time.Time {
	return t.Truncate(db.GetAvailabilityInterval())
}

// Check if two times are at least 15 minutes apart
func areAtLeastTheIntervalApart(t1, t2 time.Time) bool {
	// Calculate the absolute difference between the two times
	diff := t2.Sub(t1)

	// Check if the difference is at least 15 minutes
	return diff >= db.GetAvailabilityInterval()
}

func (s *Server) PostClients(c *gin.Context) {
	var req schema.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := s.DB.CreateClient(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

//nolint:revive
func (s *Server) GetClientsClientID(c *gin.Context, clientId openapi_types.UUID) {
	// Retrieve the user from the database
	user, err := s.DB.GetClient(clientId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Return the user details
	c.JSON(http.StatusOK, user)
}

func (s *Server) PostProviders(c *gin.Context) {
	var req schema.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := s.DB.CreateProvider(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

//nolint:revive
func (s *Server) GetProvidersProviderID(c *gin.Context, providerId openapi_types.UUID) {
	// Retrieve the user from the database
	user, err := s.DB.GetProvider(providerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Return the user details
	c.JSON(http.StatusOK, user)
}
