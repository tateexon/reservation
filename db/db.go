package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"                   // PostgreSQL driver
	"github.com/oapi-codegen/runtime/types" // Import openapi_types
	"github.com/tateexon/reservation/schema"
	"github.com/tateexon/reservation/utils"
)

const availabilityInterval = 15 * time.Minute

var avInterval time.Duration

type Database struct {
	Conn *sql.DB
}

// Initialize database connection
func NewDatabase(connStr string) (*Database, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &Database{Conn: conn}, nil
}

func GetAvailabilityInterval() time.Duration {
	if avInterval == 0 {
		if interval, ok := os.LookupEnv("AVAILABILITY_INTERVAL"); ok {
			avInterval, _ = time.ParseDuration(interval)
		} else {
			avInterval = availabilityInterval
		}
	}
	return avInterval

}

func (db *Database) GetAvailableAppointments(providerID *types.UUID, date *types.Date) ([]schema.Appointment, error) {
	var appointments []schema.Appointment

	query := `
    SELECT a.id, a.provider_id, a.start_time, a.end_time
    FROM availability a
    LEFT JOIN appointments appt ON a.provider_id = appt.provider_id AND a.start_time = appt.start_time
      AND appt.status IN ('reserved', 'confirmed')
      AND (
        appt.status = 'confirmed' OR
        (appt.status = 'reserved' AND appt.created_at > NOW() - INTERVAL '30 minutes')
      )
    WHERE appt.id IS NULL
    `

	var args []interface{}
	argIndex := 1

	if providerID != nil {
		query += fmt.Sprintf(" AND a.provider_id = $%d", argIndex)
		args = append(args, providerID.String())
		argIndex++
	}

	if date != nil {
		// We need to filter on date
		// a.start_time >= date and a.start_time < date + 1 day
		startOfDay := date.Time
		endOfDay := startOfDay.Add(24 * time.Hour)

		query += fmt.Sprintf(" AND a.start_time >= $%d AND a.start_time < $%d", argIndex, argIndex+1)
		args = append(args, startOfDay, endOfDay)
	}

	rows, err := db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var appointment schema.Appointment
		var id uuid.UUID
		var providerID uuid.UUID
		var startTime time.Time
		var endTime time.Time

		err := rows.Scan(&id, &providerID, &startTime, &endTime)
		if err != nil {
			return nil, err
		}

		appointment.Id = (*types.UUID)(&id)
		appointment.ProviderId = (*types.UUID)(&providerID)
		appointment.StartTime = &startTime
		appointment.EndTime = &endTime

		appointments = append(appointments, appointment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return appointments, nil
}

func (db *Database) GetAppointmentStartTime(availabilityID *types.UUID) (time.Time, error) {
	var startTime time.Time
	err := db.Conn.QueryRow(`
	SELECT start_time
	FROM availability
	WHERE id = $1
`, availabilityID.String()).Scan(&startTime)

	return startTime, err
}

func (db *Database) IsSlotAvailable(providerID *types.UUID, startTime *time.Time) (bool, error) {
	var count int
	err := db.Conn.QueryRow(`
        SELECT COUNT(*)
        FROM availability a
        WHERE a.provider_id = $1 AND a.start_time = $2
          AND NOT EXISTS (
            SELECT 1 FROM appointments appt
            WHERE appt.provider_id = a.provider_id
              AND appt.start_time = a.start_time
              AND appt.status IN ('reserved', 'confirmed')
              AND (
                appt.status = 'confirmed' OR
                (appt.status = 'reserved' AND appt.created_at > NOW() - INTERVAL '30 minutes')
              )
          )
    `, providerID.String(), *startTime).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *Database) ReserveAppointment(clientID, providerID *types.UUID, startTime *time.Time) (*schema.Appointment, error) {
	// Insert new appointment with status 'reserved' and current timestamp

	// First, check that the slot is still available
	available, err := db.IsSlotAvailable(providerID, startTime)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, fmt.Errorf("slot is not available")
	}

	endTime := startTime.Add(GetAvailabilityInterval())
	appointmentID := uuid.New()

	_, err = db.Conn.Exec(`
		INSERT INTO appointments (id, client_id, provider_id, start_time, end_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'reserved', NOW(), NOW())
		`, appointmentID, clientID.String(), providerID.String(), *startTime, endTime)
	if err != nil {
		return nil, err
	}
	status := schema.AppointmentStatus("reserved")
	appointment := &schema.Appointment{
		Id:         (*types.UUID)(&appointmentID),
		ClientId:   clientID,
		ProviderId: providerID,
		StartTime:  startTime,
		EndTime:    &endTime,
		Status:     &status,
	}
	return appointment, nil
}

func (db *Database) ConfirmAppointment(appointmentID types.UUID) error {
	result, err := db.Conn.Exec(`
	UPDATE appointments
	SET status = 'confirmed', updated_at = NOW()
	WHERE id = $1
	  AND status = 'reserved'
	  AND created_at > NOW() - INTERVAL '30 minutes'
`, appointmentID.String())
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Return sql.ErrNoRows to indicate not found
	}
	return nil
}

//nolint:errcheck
func (db *Database) AddAvailability(providerID types.UUID, slots []time.Time) error {
	pExists, err := db.providerExists(providerID)
	if err != nil {
		return err
	}
	if !pExists {
		return fmt.Errorf("provider does not exist")
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO availability (id, provider_id, start_time, end_time, created_at, updated_at)
	VALUES ($1, $2, $3, $4, NOW(), NOW())
	ON CONFLICT (provider_id, start_time) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, startTime := range slots {
		endTime := startTime.Add(GetAvailabilityInterval())
		availabilityID := uuid.New()
		_, err := stmt.Exec(availabilityID, providerID.String(), startTime, endTime)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) providerExists(providerID types.UUID) (bool, error) {
	var id uuid.UUID
	err := db.Conn.QueryRow(`
	SELECT id
	FROM users
	WHERE id = $1
	AND role = 'provider'
`, providerID.String()).Scan(&id)

	if err != nil {
		return false, err
	}

	return id != uuid.Nil, nil
}

func (db *Database) CreateUser(name, email, role string) (*schema.User, error) {
	userID := uuid.New()
	_, err := db.Conn.Exec(`
        INSERT INTO users (id, name, email, role, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `, userID, name, email, role)
	if err != nil {
		return nil, err
	}

	userRole := schema.UserRole(role)
	user := &schema.User{
		Id:    (*types.UUID)(&userID),
		Name:  utils.Ptr(name),
		Email: utils.Ptr(email),
		Role:  &userRole,
	}
	return user, nil
}

func (db *Database) GetUser(userID types.UUID) (*schema.User, error) {
	var user schema.User
	var id uuid.UUID
	var name, email, role string

	err := db.Conn.QueryRow(`
		SELECT id, name, email, role
		FROM users
		WHERE id = $1
	`, userID.String()).Scan(&id, &name, &email, &role)

	if err != nil {
		return nil, err
	}

	userRole := schema.UserRole(role)
	user = schema.User{
		Id:    (*types.UUID)(&id),
		Name:  utils.Ptr(name),
		Email: utils.Ptr(email),
		Role:  &userRole,
	}

	return &user, nil
}
