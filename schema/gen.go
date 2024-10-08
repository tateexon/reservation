// Package schema provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version 2.4.0 DO NOT EDIT.
package schema

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Defines values for AppointmentStatus.
const (
	Confirmed AppointmentStatus = "confirmed"
	Reserved  AppointmentStatus = "reserved"
)

// Defines values for CreateUserRequestRole.
const (
	CreateUserRequestRoleClient   CreateUserRequestRole = "client"
	CreateUserRequestRoleProvider CreateUserRequestRole = "provider"
)

// Defines values for UserRole.
const (
	UserRoleClient   UserRole = "client"
	UserRoleProvider UserRole = "provider"
)

// Appointment defines model for Appointment.
type Appointment struct {
	ClientId   *openapi_types.UUID `json:"client_id,omitempty"`
	EndTime    *time.Time          `json:"end_time,omitempty"`
	Id         *openapi_types.UUID `json:"id,omitempty"`
	ProviderId *openapi_types.UUID `json:"provider_id,omitempty"`
	StartTime  *time.Time          `json:"start_time,omitempty"`
	Status     *AppointmentStatus  `json:"status,omitempty"`
}

// AppointmentStatus defines model for Appointment.Status.
type AppointmentStatus string

// Availability defines model for Availability.
type Availability struct {
	EndTime    *time.Time          `json:"end_time,omitempty"`
	Id         *openapi_types.UUID `json:"id,omitempty"`
	ProviderId *openapi_types.UUID `json:"provider_id,omitempty"`
	StartTime  *time.Time          `json:"start_time,omitempty"`
}

// CreateUserRequest defines model for CreateUserRequest.
type CreateUserRequest struct {
	Email string                `json:"email"`
	Name  string                `json:"name"`
	Role  CreateUserRequestRole `json:"role"`
}

// CreateUserRequestRole defines model for CreateUserRequest.Role.
type CreateUserRequestRole string

// User defines model for User.
type User struct {
	Email *string             `json:"email,omitempty"`
	Id    *openapi_types.UUID `json:"id,omitempty"`
	Name  *string             `json:"name,omitempty"`
	Role  *UserRole           `json:"role,omitempty"`
}

// UserRole defines model for User.Role.
type UserRole string

// GetAppointmentsParams defines parameters for GetAppointments.
type GetAppointmentsParams struct {
	ProviderId *openapi_types.UUID `form:"providerId,omitempty" json:"providerId,omitempty"`
	Date       *openapi_types.Date `form:"date,omitempty" json:"date,omitempty"`
}

// PostAppointmentsJSONBody defines parameters for PostAppointments.
type PostAppointmentsJSONBody struct {
	AvailabilityId *openapi_types.UUID `json:"availability_id,omitempty"`
	ClientId       *openapi_types.UUID `json:"client_id,omitempty"`
	ProviderId     *openapi_types.UUID `json:"provider_id,omitempty"`
}

// PostAppointmentsJSONRequestBody defines body for PostAppointments for application/json ContentType.
type PostAppointmentsJSONRequestBody PostAppointmentsJSONBody

// PostProvidersProviderIdAvailabilityJSONRequestBody defines body for PostProvidersProviderIdAvailability for application/json ContentType.
type PostProvidersProviderIdAvailabilityJSONRequestBody = Availability

// PostUsersJSONRequestBody defines body for PostUsers for application/json ContentType.
type PostUsersJSONRequestBody = CreateUserRequest

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Get available appointment slots
	// (GET /appointments)
	GetAppointments(c *gin.Context, params GetAppointmentsParams)
	// Reserve an appointment slot
	// (POST /appointments)
	PostAppointments(c *gin.Context)
	// Confirm a reservation
	// (POST /appointments/{appointmentId}/confirm)
	PostAppointmentsAppointmentIdConfirm(c *gin.Context, appointmentId openapi_types.UUID)
	// Submit provider availability
	// (POST /providers/{providerId}/availability)
	PostProvidersProviderIdAvailability(c *gin.Context, providerId openapi_types.UUID)
	// Create a new user (client or provider)
	// (POST /users)
	PostUsers(c *gin.Context)
	// Get user details
	// (GET /users/{userId})
	GetUsersUserId(c *gin.Context, userId openapi_types.UUID)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandler       func(*gin.Context, error, int)
}

type MiddlewareFunc func(c *gin.Context)

// GetAppointments operation middleware
func (siw *ServerInterfaceWrapper) GetAppointments(c *gin.Context) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetAppointmentsParams

	// ------------- Optional query parameter "providerId" -------------

	err = runtime.BindQueryParameter("form", true, false, "providerId", c.Request.URL.Query(), &params.ProviderId)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter providerId: %w", err), http.StatusBadRequest)
		return
	}

	// ------------- Optional query parameter "date" -------------

	err = runtime.BindQueryParameter("form", true, false, "date", c.Request.URL.Query(), &params.Date)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter date: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetAppointments(c, params)
}

// PostAppointments operation middleware
func (siw *ServerInterfaceWrapper) PostAppointments(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.PostAppointments(c)
}

// PostAppointmentsAppointmentIdConfirm operation middleware
func (siw *ServerInterfaceWrapper) PostAppointmentsAppointmentIdConfirm(c *gin.Context) {

	var err error

	// ------------- Path parameter "appointmentId" -------------
	var appointmentId openapi_types.UUID

	err = runtime.BindStyledParameterWithOptions("simple", "appointmentId", c.Param("appointmentId"), &appointmentId, runtime.BindStyledParameterOptions{Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter appointmentId: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.PostAppointmentsAppointmentIdConfirm(c, appointmentId)
}

// PostProvidersProviderIdAvailability operation middleware
func (siw *ServerInterfaceWrapper) PostProvidersProviderIdAvailability(c *gin.Context) {

	var err error

	// ------------- Path parameter "providerId" -------------
	var providerId openapi_types.UUID

	err = runtime.BindStyledParameterWithOptions("simple", "providerId", c.Param("providerId"), &providerId, runtime.BindStyledParameterOptions{Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter providerId: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.PostProvidersProviderIdAvailability(c, providerId)
}

// PostUsers operation middleware
func (siw *ServerInterfaceWrapper) PostUsers(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.PostUsers(c)
}

// GetUsersUserId operation middleware
func (siw *ServerInterfaceWrapper) GetUsersUserId(c *gin.Context) {

	var err error

	// ------------- Path parameter "userId" -------------
	var userId openapi_types.UUID

	err = runtime.BindStyledParameterWithOptions("simple", "userId", c.Param("userId"), &userId, runtime.BindStyledParameterOptions{Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter userId: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetUsersUserId(c, userId)
}

// GinServerOptions provides options for the Gin server.
type GinServerOptions struct {
	BaseURL      string
	Middlewares  []MiddlewareFunc
	ErrorHandler func(*gin.Context, error, int)
}

// RegisterHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterHandlers(router gin.IRouter, si ServerInterface) {
	RegisterHandlersWithOptions(router, si, GinServerOptions{})
}

// RegisterHandlersWithOptions creates http.Handler with additional options
func RegisterHandlersWithOptions(router gin.IRouter, si ServerInterface, options GinServerOptions) {
	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"msg": err.Error()})
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandler:       errorHandler,
	}

	router.GET(options.BaseURL+"/appointments", wrapper.GetAppointments)
	router.POST(options.BaseURL+"/appointments", wrapper.PostAppointments)
	router.POST(options.BaseURL+"/appointments/:appointmentId/confirm", wrapper.PostAppointmentsAppointmentIdConfirm)
	router.POST(options.BaseURL+"/providers/:providerId/availability", wrapper.PostProvidersProviderIdAvailability)
	router.POST(options.BaseURL+"/users", wrapper.PostUsers)
	router.GET(options.BaseURL+"/users/:userId", wrapper.GetUsersUserId)
}
