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
	providerID := params.ProviderId
	date := params.Date

	// Get available appointment slots from the database
	slots, err := s.DB.GetAvailableAppointments(providerID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch appointments"})
		return
	}

	c.JSON(http.StatusOK, slots)
}

func (s *Server) PostAppointments(c *gin.Context) {
	var req schema.PostAppointmentsJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// get appointment
	startTime, err := s.DB.GetAppointmentStartTime(req.AvailabilityId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid availability id"})
		return
	}

	// Business logic checks
	if !s.isReservationAtLeast24HoursInAdvance(&startTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reservations must be made at least 24 hours in advance"})
		return
	}

	// Check if the slot is available
	available, err := s.DB.IsSlotAvailable(req.ProviderId, &startTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check slot availability"})
		return
	}
	if !available {
		c.JSON(http.StatusConflict, gin.H{"error": "Slot is not available"})
		return
	}

	// Reserve the appointment
	appointment, err := s.DB.ReserveAppointment(req.ClientId, req.ProviderId, &startTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reserve appointment"})
		return
	}

	c.JSON(http.StatusCreated, appointment)
}

func (s *Server) isReservationAtLeast24HoursInAdvance(startTime *time.Time) bool {
	return time.Until(*startTime) >= 24*time.Hour
}

//nolint:revive
func (s *Server) PostAppointmentsAppointmentIdConfirm(c *gin.Context, appointmentId openapi_types.UUID) {
	// Confirm the reservation
	err := s.DB.ConfirmAppointment(appointmentId)
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

//nolint:revive
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
	if endTime.Before(startTime) || !areAtLeastTheIntervalApart(startTime, endTime) {
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

func (s *Server) PostUsers(c *gin.Context) {
	var req schema.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := s.DB.CreateUser(req.Name, req.Email, string(req.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

//nolint:revive
func (s *Server) GetUsersUserId(c *gin.Context, userId openapi_types.UUID) {
	// Retrieve the user from the database
	user, err := s.DB.GetUser(userId)
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
