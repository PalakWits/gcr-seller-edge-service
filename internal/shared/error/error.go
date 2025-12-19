package error

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type CustomError struct {
	Message  string `json:"message"`
	Code     string `json:"code"`
	HTTPCode int    `json:"httpCode"`
	Details  any    `json:"details,omitempty"`
}

func (err *CustomError) Error() string {
	if err.Code != "" {
		return fmt.Sprintf("[%s] %s", err.Code, err.Message)
	}
	return err.Message
}

func (err *CustomError) Is(target error) bool {
	if targetErr, ok := target.(*CustomError); ok {
		return err.Code == targetErr.Code && err.Message == targetErr.Message && err.HTTPCode == targetErr.HTTPCode
	}
	return false
}

func NewCustomError(httpCode int, code, message string, details ...any) *CustomError {
	err := &CustomError{
		HTTPCode: httpCode,
		Code:     code,
		Message:  message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

var (
	ErrUserNotFound       = NewCustomError(404, "USER_2001", "User not found")
	ErrInvalidAPIKey      = NewCustomError(401, "USER_2002", "Invalid API key")
	ErrUserNotActive      = NewCustomError(403, "USER_2003", "User is not active")
	ErrDuplicateUser      = NewCustomError(409, "USER_2004", "User with this name already exists")
	ErrFailedToCreateUser = NewCustomError(500, "USER_2005", "Failed to create user")
	ErrFailedToUpdateUser = NewCustomError(500, "USER_2006", "Failed to update user")
	ErrFailedToDeleteUser = NewCustomError(500, "USER_2007", "Failed to delete user")
	ErrFailedToGetUser    = NewCustomError(500, "USER_2008", "Failed to retrieve user")

	ErrXStudioConnectionFailed = NewCustomError(503, "XSTUDIO_2001", "Failed to connect to XStudio service")
	ErrXStudioRequestFailed    = NewCustomError(502, "XSTUDIO_2002", "XStudio request failed")
	ErrXStudioInvalidResponse  = NewCustomError(502, "XSTUDIO_2003", "Invalid response from XStudio")
	ErrXStudioTimeout          = NewCustomError(504, "XSTUDIO_2004", "XStudio request timeout")

	ErrMissingAPIKey       = NewCustomError(401, "AUTH_2001", "API key is required")
	ErrInvalidAPIKeyFormat = NewCustomError(400, "AUTH_2002", "Invalid API key format")

	ErrDatabaseConnectionFailed  = NewCustomError(500, "DB_2001", "Failed to connect to database")
	ErrDatabaseQueryFailed       = NewCustomError(500, "DB_2002", "Database query failed")
	ErrDatabaseTransactionFailed = NewCustomError(500, "DB_2003", "Database transaction failed")

	ErrInvalidRequestBody   = NewCustomError(400, "REQUEST_2001", "Invalid request body")
	ErrMissingRequiredField = NewCustomError(400, "REQUEST_2002", "Missing required field")
	ErrInvalidFieldFormat   = NewCustomError(400, "REQUEST_2003", "Invalid field format")

	ErrHTTPBadRequest         = NewCustomError(400, "HTTP_400", "Bad Request")
	ErrHTTPUnauthorized       = NewCustomError(401, "HTTP_401", "Unauthorized")
	ErrHTTPForbidden          = NewCustomError(403, "HTTP_403", "Forbidden")
	ErrHTTPNotFound           = NewCustomError(404, "HTTP_404", "Not Found")
	ErrHTTPConflict           = NewCustomError(409, "HTTP_409", "Conflict")
	ErrHTTPInternalServer     = NewCustomError(500, "HTTP_500", "Internal Server Error")
	ErrHTTPServiceUnavailable = NewCustomError(503, "HTTP_503", "Service Unavailable")
	ErrHTTPGatewayTimeout     = NewCustomError(504, "HTTP_504", "Gateway Timeout")
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := c.Locals("request_id")

		if customErr, ok := err.(*CustomError); ok {
			response := fiber.Map{
				"error":   customErr.Message,
				"code":    customErr.Code,
				"details": customErr.Details,
			}
			if requestID != nil {
				response["request_id"] = requestID
			}
			return c.Status(customErr.HTTPCode).JSON(response)
		}

		if fiberErr, ok := err.(*fiber.Error); ok {
			response := fiber.Map{
				"error": fiberErr.Message,
			}
			if requestID != nil {
				response["request_id"] = requestID
			}
			return c.Status(fiberErr.Code).JSON(response)
		}

		response := fiber.Map{
			"error": "Internal server error",
		}
		if requestID != nil {
			response["request_id"] = requestID
		}
		return c.Status(fiber.StatusInternalServerError).JSON(response)
	}
}
