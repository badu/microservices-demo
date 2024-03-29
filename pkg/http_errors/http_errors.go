package httpErrors

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrBadRequest       = "Bad request"
	ErrAlreadyExists    = "Already exists"
	ErrNoSuchUser       = "User not found"
	ErrWrongCredentials = "Wrong Credentials"
	ErrNotFound         = "Not Found"
	ErrUnauthorized     = "Unauthorized"
	ErrForbidden        = "Forbidden"
	ErrBadQueryParams   = "Invalid query params"
	ErrRequestTimeout   = "Request Timeout"
	ErrInvalidEmail     = "Invalid email"
	ErrInvalidPassword  = "Invalid password"
	ErrInvalidField     = "Invalid field"
)

var (
	BadRequest            = Error("bad request")
	WrongCredentials      = Error("wrong Credentials")
	NotFound              = Error("not Found")
	Unauthorized          = Error("unauthorized")
	Forbidden             = Error("forbidden")
	PermissionDenied      = Error("permission Denied")
	ExpiredCSRFError      = Error("expired CSRF token")
	WrongCSRFToken        = Error("wrong CSRF token")
	CSRFNotPresented      = Error("CSRF not presented")
	NotRequiredFields     = Error("no such required fields")
	BadQueryParams        = Error("invalid query params")
	InternalServerError   = Error("internal Server Error")
	RequestTimeoutError   = Error("request Timeout")
	ExistsEmailError      = Error("user with given email already exists")
	InvalidJWTToken       = Error("invalid JWT token")
	InvalidJWTClaims      = Error("invalid JWT claims")
	NotAllowedImageHeader = Error("not allowed image header")
	NoCookie              = Error("not found cookie header")
	InvalidUUID           = Error("invalid uuid")
)

// Rest error interface
type RestErr interface {
	Status() int
	Error() string
	Causes() interface{}
	ErrBody() RestError
}

// Rest error struct
type RestError struct {
	ErrCauses interface{} `json:"err_causes,omitempty"`
	ErrError  string      `json:"error,omitempty"`
	ErrStatus int         `json:"status,omitempty"`
}

// Error body
func (e RestError) ErrBody() RestError {
	return e
}

// Error  Error() interface method
func (e RestError) Error() string {
	return fmt.Sprintf("status: %d - errors: %s - causes: %v", e.ErrStatus, e.ErrError, e.ErrCauses)
}

// Error status
func (e RestError) Status() int {
	return e.ErrStatus
}

// RestError Causes
func (e RestError) Causes() interface{} {
	return e.ErrCauses
}

// New Rest Error
func NewRestError(status int, err string, causes interface{}) RestErr {
	return RestError{
		ErrStatus: status,
		ErrError:  err,
		ErrCauses: causes,
	}
}

// New Rest Error With Message
func NewRestErrorWithMessage(status int, err string, causes interface{}) RestErr {
	return RestError{
		ErrStatus: status,
		ErrError:  err,
		ErrCauses: causes,
	}
}

// New Rest Error From Bytes
func NewRestErrorFromBytes(bytes []byte) (RestErr, error) {
	var apiErr RestError
	if err := json.Unmarshal(bytes, &apiErr); err != nil {
		return nil, errors.New("invalid json")
	}
	return apiErr, nil
}

// New Bad Request Error
func NewBadRequestError(causes interface{}) RestErr {
	return RestError{
		ErrStatus: http.StatusBadRequest,
		ErrError:  BadRequest.Error(),
		ErrCauses: causes,
	}
}

// New Not Found Error
func NewNotFoundError(causes interface{}) RestErr {
	return RestError{
		ErrStatus: http.StatusNotFound,
		ErrError:  NotFound.Error(),
		ErrCauses: causes,
	}
}

// New Unauthorized Error
func NewUnauthorizedError(causes interface{}) RestErr {
	return RestError{
		ErrStatus: http.StatusUnauthorized,
		ErrError:  Unauthorized.Error(),
		ErrCauses: causes,
	}
}

// New Forbidden Error
func NewForbiddenError(causes interface{}) RestErr {
	return RestError{
		ErrStatus: http.StatusForbidden,
		ErrError:  Forbidden.Error(),
		ErrCauses: causes,
	}
}

// New Internal Server Error
func NewInternalServerError(causes interface{}) RestErr {
	result := RestError{
		ErrStatus: http.StatusInternalServerError,
		ErrError:  InternalServerError.Error(),
		ErrCauses: causes,
	}
	return result
}

// Parser of error string messages returns RestError
func ParseErrors(err error) RestErr {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return NewRestError(http.StatusNotFound, ErrNotFound, nil)
	case errors.Is(err, context.DeadlineExceeded):
		return NewRestError(http.StatusRequestTimeout, ErrRequestTimeout, nil)
	case errors.Is(err, Unauthorized):
		return NewRestError(http.StatusUnauthorized, ErrUnauthorized, nil)
	case errors.Is(err, WrongCredentials):
		return NewRestError(http.StatusUnauthorized, ErrUnauthorized, nil)
	case strings.Contains(strings.ToLower(err.Error()), "sqlstate"):
		return parseSqlErrors(err)
	case strings.Contains(strings.ToLower(err.Error()), "field validation"):
		return parseValidatorError(err)
	case strings.Contains(strings.ToLower(err.Error()), "unmarshal"):
		return NewRestError(http.StatusBadRequest, ErrBadRequest, err)
	case strings.Contains(strings.ToLower(err.Error()), "uuid"):
		return NewRestError(http.StatusBadRequest, ErrBadRequest, err)
	case strings.Contains(strings.ToLower(err.Error()), "cookie"):
		return NewRestError(http.StatusUnauthorized, ErrUnauthorized, err)
	case strings.Contains(strings.ToLower(err.Error()), "token"):
		return NewRestError(http.StatusUnauthorized, ErrUnauthorized, err)
	case strings.Contains(strings.ToLower(err.Error()), "bcrypt"):
		return NewRestError(http.StatusBadRequest, ErrBadRequest, nil)
	default:
		if restErr, ok := err.(RestErr); ok {
			return restErr
		}
		return NewInternalServerError(err)
	}
}

func parseSqlErrors(err error) RestErr {
	return NewRestError(http.StatusBadRequest, ErrBadRequest, err)
}

func parseValidatorError(err error) RestErr {
	if strings.Contains(err.Error(), "Password") {
		return NewRestError(http.StatusBadRequest, ErrInvalidPassword, err)
	}

	if strings.Contains(err.Error(), "Email") {
		return NewRestError(http.StatusBadRequest, ErrInvalidEmail, err)
	}

	return NewRestError(http.StatusBadRequest, ErrInvalidField, err)
}

// Error response
func ErrorResponse(err error) (int, interface{}) {
	return ParseErrors(err).Status(), ParseErrors(err)
}

// Error response object and status code
func ErrorCtxResponse(ctx echo.Context, err error) error {
	restErr := ParseErrors(err)
	return ctx.JSON(restErr.Status(), restErr.ErrBody())
}
