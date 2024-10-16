package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/tateexon/reservation/schema"
	"github.com/tateexon/reservation/utils"
)

const (
	dbname   = "yourdb"
	user     = "youruser"
	password = "yourpassword"
)

func startTestDatabase(t *testing.T) *Database {
	ctx := context.Background()

	ctr := utils.StartTestPostgres(ctx, t, dbname, user, password)

	// explicitly set sslmode=disable because the container is not configured to use TLS
	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dbInstance, err := NewDatabase(connStr)
	require.NoError(t, err, "Failed to connect to the database")

	return dbInstance
}

// Helper functions to create test data
func createTestProvider(t *testing.T, dbInstance *Database) *types.UUID {
	providerID := uuid.New()
	_, err := dbInstance.Conn.Exec(`
        INSERT INTO users (id, name, email, role, created_at, updated_at)
        VALUES ($1, $2, $3, 'provider', NOW(), NOW())
    `, providerID, "Test Provider", fmt.Sprintf("provider-%s@example.com", providerID.String()))
	require.NoError(t, err)
	return (*types.UUID)(&providerID)
}

func createTestClient(t *testing.T, db *Database) *types.UUID {
	clientID := uuid.New()
	_, err := db.Conn.Exec(`
        INSERT INTO users (id, name, email, role, created_at, updated_at)
        VALUES ($1, $2, $3, 'client', NOW(), NOW())
    `, clientID, "Test Client", fmt.Sprintf("client-%s@example.com", clientID.String()))
	require.NoError(t, err)
	return (*types.UUID)(&clientID)
}

func addTestAvailability(t *testing.T, dbInstance *Database, providerID *types.UUID, slots []time.Time) {
	err := dbInstance.AddAvailability(*providerID, slots)
	require.NoError(t, err)
}

func TestStartDatabase(t *testing.T) {
	t.Parallel()
	startTestDatabase(t)
	// time.Sleep(2 * time.Minute)
}

func TestAddAvailability(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	providerID := createTestProvider(t, dbInstance)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	slots := utils.GenerateTimeSlots(startTime, startTime.Add(2*time.Hour), GetAvailabilityInterval())

	err := dbInstance.AddAvailability(*providerID, slots)
	require.NoError(t, err)

	// Verify slots were added
	var count int
	err = dbInstance.Conn.QueryRow(`
        SELECT COUNT(*) FROM availability WHERE provider_id = $1
    `, providerID.String()).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, len(slots), count)
}

func TestIsSlotAvailable(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	providerID := createTestProvider(t, dbInstance)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}

	addTestAvailability(t, dbInstance, providerID, slots)

	available, err := dbInstance.IsSlotAvailable(providerID, &startTime)
	require.NoError(t, err)
	require.True(t, available)

	// Reserve the slot
	clientID := createTestClient(t, dbInstance)
	_, err = dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)

	// Check availability again
	available, err = dbInstance.IsSlotAvailable(providerID, &startTime)
	require.NoError(t, err)
	require.False(t, available)
}

func TestReserveAppointment(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}

	addTestAvailability(t, dbInstance, providerID, slots)

	appointment, err := dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)
	require.NotNil(t, appointment)
	require.Equal(t, schema.AppointmentStatus("reserved"), *appointment.Status)

	// Attempt to reserve the same slot again
	_, err = dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.Error(t, err)
}

func TestConfirmAppointment(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}

	addTestAvailability(t, dbInstance, providerID, slots)

	appointment, err := dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)

	// Confirm the appointment
	err = dbInstance.ConfirmAppointment(*appointment.Id)
	require.NoError(t, err)

	// Verify status is updated
	var status string
	err = dbInstance.Conn.QueryRow(`
        SELECT status FROM appointments WHERE id = $1
    `, appointment.Id.String()).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "confirmed", status)
}

func TestGetAvailableAppointments(t *testing.T) {
	t.Parallel()
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	providerID := createTestProvider(t, dbInstance)
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	slots := utils.GenerateTimeSlots(startTime, startTime.Add(2*time.Hour), GetAvailabilityInterval())

	addTestAvailability(t, dbInstance, providerID, slots)

	// Initially, all slots should be available
	appointments, err := dbInstance.GetAvailableAppointments(providerID, nil)
	require.NoError(t, err)
	require.Equal(t, len(slots), len(appointments))

	// Reserve a slot
	clientID := createTestClient(t, dbInstance)
	_, err = dbInstance.ReserveAppointment(clientID, providerID, &slots[0])
	require.NoError(t, err)

	// Now, one slot should be unavailable
	appointments, err = dbInstance.GetAvailableAppointments(providerID, nil)
	require.NoError(t, err)
	require.Equal(t, len(slots)-1, len(appointments))
}

func TestReservationExpiryLogic(t *testing.T) {
	// Start test database and server setup
	dbInstance := startTestDatabase(t)
	defer dbInstance.Conn.Close()

	// Create a test provider and client
	providerID := createTestProvider(t, dbInstance)
	clientID := createTestClient(t, dbInstance)

	// Add availability for the provider
	startTime := time.Now().Add(25 * time.Hour).Truncate(time.Minute)
	slots := []time.Time{startTime}
	addTestAvailability(t, dbInstance, providerID, slots)

	// Reserve the appointment
	appointment, err := dbInstance.ReserveAppointment(clientID, providerID, &startTime)
	require.NoError(t, err)
	require.NotNil(t, appointment)

	// Simulate passage of time by updating the 'created_at' to more than 30 minutes ago
	_, err = dbInstance.Conn.Exec(`
        UPDATE appointments
        SET created_at = created_at - INTERVAL '31 minutes'
        WHERE id = $1
    `, appointment.Id.String())
	require.NoError(t, err)

	// Check if the slot is now available
	available, err := dbInstance.IsSlotAvailable(providerID, &startTime)
	require.NoError(t, err)
	require.True(t, available, "Slot should be available after reservation has expired")

	// Attempt to reserve the slot again
	clientID2 := createTestClient(t, dbInstance)
	appointment2, err := dbInstance.ReserveAppointment(clientID2, providerID, &startTime)
	require.NoError(t, err)
	require.NotNil(t, appointment2)

	// Verify that the new reservation is successful
	require.Equal(t, *appointment2.StartTime, *appointment.StartTime)
	require.NotEqual(t, appointment2.ClientId.String(), appointment.ClientId.String())
}
