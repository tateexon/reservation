package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/tateexon/reservation/db"
	"github.com/tateexon/reservation/schema"
	"github.com/tateexon/reservation/utils"
)

const (
	dbname   = "yourdb"
	user     = "youruser"
	password = "yourpassword"
)

func startTestDatabase(t *testing.T) *db.Database {
	ctx := context.Background()

	ctr := utils.StartTestPostgres(ctx, t, dbname, user, password)

	// explicitly set sslmode=disable because the container is not configured to use TLS
	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dbInstance, err := db.NewDatabase(connStr)
	require.NoError(t, err, "Failed to connect to the database")

	return dbInstance
}

// Helper functions to create test data
func createTestProvider(t *testing.T, dbInstance *db.Database) *types.UUID {
	providerID := uuid.New()
	_, err := dbInstance.Conn.Exec(`
        INSERT INTO users (id, name, email, role, created_at, updated_at)
        VALUES ($1, $2, $3, 'provider', NOW(), NOW())
    `, providerID, "Test Provider", fmt.Sprintf("provider-%s@example.com", providerID.String()))
	require.NoError(t, err)
	return (*types.UUID)(&providerID)
}

func createTestClient(t *testing.T, db *db.Database) *types.UUID {
	clientID := uuid.New()
	_, err := db.Conn.Exec(`
        INSERT INTO users (id, name, email, role, created_at, updated_at)
        VALUES ($1, $2, $3, 'client', NOW(), NOW())
    `, clientID, "Test Client", fmt.Sprintf("client-%s@example.com", clientID.String()))
	require.NoError(t, err)
	return (*types.UUID)(&clientID)
}

func addTestAvailability(t *testing.T, dbInstance *db.Database, providerID *types.UUID, slots []time.Time) {
	err := dbInstance.AddAvailability(*providerID, slots)
	require.NoError(t, err)
}

func setupTestServer(dbInstance *db.Database) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	server := &Server{DB: dbInstance}
	schema.RegisterHandlers(router, server)

	return router
}

func TestPostProvidersProviderIdAvailability(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)

	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	availability := schema.Availability{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	reqBody, err := json.Marshal(availability)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/providers/"+providerID.String()+"/availability", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Availability added", response["message"])

	// Verify that availability slots were added to the database
	var count int
	err = dbInstance.Conn.QueryRow(`
        SELECT COUNT(*) FROM availability WHERE provider_id = $1
    `, providerID.String()).Scan(&count)
	require.NoError(t, err)
	expectedSlots := utils.GenerateTimeSlots(startTime, endTime, db.GetAvailabilityInterval())
	require.Equal(t, len(expectedSlots), count)
}

func TestPostProvidersProviderIdAvailability_InvalidProviderId(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	createTestProvider(t, dbInstance)

	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	availability := schema.Availability{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	reqBody, err := json.Marshal(availability)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/providers/"+uuid.NewString()+"/availability", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Failed to add availability", response["error"])
}

func TestGetAppointments(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	// Create a test provider and add availability
	providerID := createTestProvider(t, dbInstance)

	// Get the time zone offset in seconds from UTC
	_, offsetSeconds := time.Now().Zone()

	// Convert the offset from seconds to a time.Duration
	offsetDuration := time.Duration(offsetSeconds) * time.Second

	startTime := time.Now().Add(72 * time.Hour).Truncate(24 * time.Hour).Add(offsetDuration)
	fmt.Println(startTime)
	endTime := startTime.Add(2 * time.Hour)
	fmt.Println(endTime)
	slots := utils.GenerateTimeSlots(startTime, endTime, db.GetAvailabilityInterval())
	addTestAvailability(t, dbInstance, providerID, slots)

	req, err := http.NewRequest(http.MethodGet, "/appointments?providerId="+providerID.String(), nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var appointments []schema.Appointment
	err = json.Unmarshal(w.Body.Bytes(), &appointments)
	require.NoError(t, err)
	require.Equal(t, 8, len(appointments))

	filterDate := startTime.Format("2006-01-02")
	fmt.Println(filterDate)

	url := fmt.Sprintf("/appointments?providerId=%s&date=%s", providerID.String(), filterDate)
	req, err = http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &appointments)
	require.NoError(t, err)
	require.Equal(t, 8, len(appointments))
}

func TestGetAppointments_WithDateFilter(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)

	// Define two different dates
	date1 := time.Now().Add(48 * time.Hour).Truncate(time.Hour * 24).Add(1 * time.Second)

	// Add availability for both dates, aka 32 open slots
	startTime1 := date1.Add(9 * time.Hour) // 9 AM on date1
	fmt.Println(startTime1)
	endTime1 := date1.Add(17 * time.Hour) // 5 PM on date1
	fmt.Println(endTime1)
	slots1 := utils.GenerateTimeSlots(startTime1, endTime1, db.GetAvailabilityInterval())
	addTestAvailability(t, dbInstance, providerID, slots1)

	// Define the date to filter
	filterDate := startTime1.Format("2006-01-02") // YYYY-MM-DD format
	fmt.Println(filterDate)

	// Create HTTP GET request with date filter
	url := fmt.Sprintf("/appointments?providerId=%s&date=%s", providerID.String(), filterDate)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	require.Equal(t, http.StatusOK, w.Code)

	var appointments []schema.Appointment
	err = json.Unmarshal(w.Body.Bytes(), &appointments)
	require.NoError(t, err)

	expectedAvailableSlots := len(slots1)

	// Verify the number of available slots
	require.Equal(t, expectedAvailableSlots, len(appointments), "Number of available appointments should match expected slots for the date")

	// verify that all returned slots are indeed on the specified date
	for _, appt := range appointments {
		require.Equal(t, startTime1.Year(), appt.StartTime.Year())
		require.Equal(t, startTime1.Month(), appt.StartTime.Month())
		require.Equal(t, startTime1.Day(), appt.StartTime.Day())
		require.Equal(t, providerID.String(), appt.ProviderId.String())
	}
}

func TestPostAppointments(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)

	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Minute) // Ensure it's more than 24 hours in advance
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)

	appointments, err := dbInstance.GetAvailableAppointments(providerID, &types.Date{Time: startTime})
	require.NoError(t, err)
	require.True(t, len(appointments) > 0)

	// Prepare the request body
	appointmentReq := schema.PostAppointmentsJSONRequestBody{
		ClientId:       clientID,
		ProviderId:     providerID,
		AvailabilityId: appointments[0].Id,
	}

	reqBody, err := json.Marshal(appointmentReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/appointments", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var appointment schema.Appointment
	err = json.Unmarshal(w.Body.Bytes(), &appointment)
	require.NoError(t, err)
	require.Equal(t, schema.AppointmentStatus("reserved"), *appointment.Status)

	// Verify that the appointment was added to the database
	var status string
	err = dbInstance.Conn.QueryRow(`
        SELECT status FROM appointments WHERE id = $1
    `, appointment.Id.String()).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "reserved", status)
}

func TestPostAppointmentsAppointmentIdConfirm(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)

	// Add availability and reserve an appointment
	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)
	appointment, err := dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/appointments/"+appointment.Id.String()+"/confirm", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Appointment confirmed", response["message"])

	// Verify that the appointment status is updated in the database
	var status string
	err = dbInstance.Conn.QueryRow(`
        SELECT status FROM appointments WHERE id = $1
    `, appointment.Id.String()).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "confirmed", status)
}

func TestPostAppointments_LessThan24HoursInAdvance(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)

	startTime := time.Now().Add(2 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)

	appointments, err := dbInstance.GetAvailableAppointments(providerID, &types.Date{Time: startTime})
	require.NoError(t, err)
	require.True(t, len(appointments) > 0)

	// Prepare the request body
	appointmentReq := schema.PostAppointmentsJSONRequestBody{
		ClientId:       clientID,
		ProviderId:     providerID,
		AvailabilityId: appointments[0].Id,
	}

	reqBody, err := json.Marshal(appointmentReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/appointments", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Reservations must be made at least 24 hours in advance", response["error"])
}

func TestPostAppointments_SlotUnavailable(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()
	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)
	clientID2 := createTestClient(t, dbInstance) // Second client attempting to reserve

	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)

	appointments, err := dbInstance.GetAvailableAppointments(providerID, &types.Date{Time: startTime})
	require.NoError(t, err)
	require.True(t, len(appointments) > 0)

	// Prepare the request body for the first reservation
	appointmentReq := schema.PostAppointmentsJSONRequestBody{
		ClientId:       clientID,
		ProviderId:     providerID,
		AvailabilityId: appointments[0].Id,
	}
	reqBody, err := json.Marshal(appointmentReq)
	require.NoError(t, err)

	// Create HTTP request for the first reservation
	req, err := http.NewRequest(http.MethodPost, "/appointments", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Perform the first reservation request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Prepare the request body for the second reservation attempt
	appointmentReq2 := schema.PostAppointmentsJSONRequestBody{
		ClientId:       clientID2,
		ProviderId:     providerID,
		AvailabilityId: appointments[0].Id,
	}
	reqBody2, err := json.Marshal(appointmentReq2)
	require.NoError(t, err)

	// Create HTTP request for the second reservation
	req2, err := http.NewRequest(http.MethodPost, "/appointments", bytes.NewBuffer(reqBody2))
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")

	// Perform the second reservation request
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusConflict, w2.Code)
	var response map[string]string
	err = json.Unmarshal(w2.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Slot is not available", response["error"])
}

func TestPostAppointmentsAppointmentIdConfirm_NonExistent(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()
	router := setupTestServer(dbInstance)

	// Generate a random appointment ID that doesn't exist
	invalidAppointmentID := uuid.New()

	req, err := http.NewRequest(http.MethodPost, "/appointments/"+invalidAppointmentID.String()+"/confirm", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Appointment not found or may have expired", response["error"])
}

func TestPostProvidersProviderIdAvailability_InvalidTimeRange(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()
	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)

	// Prepare the request body with invalid time range (end time before start time)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	endTime := startTime.Add(-2 * time.Hour) // End time is before start time
	availability := schema.Availability{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	reqBody, err := json.Marshal(availability)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/providers/"+providerID.String()+"/availability", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "End time must be after start time and at least 15 minutes apart", response["error"])
}

func TestConfirmAppointment_ExpiredReservation(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()
	router := setupTestServer(dbInstance)

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)

	// Add availability and reserve an appointment
	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)
	appointment, err := dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)

	// Simulate passage of time to expire the reservation
	_, err = dbInstance.Conn.Exec(`
        UPDATE appointments
        SET created_at = created_at - INTERVAL '31 minutes'
        WHERE id = $1
    `, appointment.Id.String())
	require.NoError(t, err)

	// Attempt to confirm the expired reservation
	req, err := http.NewRequest(http.MethodPost, "/appointments/"+appointment.Id.String()+"/confirm", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Appointment not found or may have expired", response["error"])
}
