package utils

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeHasura           ErrorCode = "HASURA_ERROR"
	ErrCodeServer           ErrorCode = "SERVER_ERROR"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
)

type ServiceError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return e.Message
}

func NewValidationError(message, details string) *ServiceError {
	return &ServiceError{Code: ErrCodeValidation, Message: message, Details: details}
}

func NewHasuraError(message, details string) *ServiceError {
	return &ServiceError{Code: ErrCodeHasura, Message: message, Details: details}
}

func NewServerError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeServer, Message: message}
}

func NewNotFoundError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeNotFound, Message: message}
}

func NewBadRequestError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeBadRequest, Message: message}
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Code    ErrorCode   `json:"code"`
	Details string      `json:"details,omitempty"`
}

func WriteJSONError(w http.ResponseWriter, err *ServiceError, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error:   err.Message,
		Code:    err.Code,
		Details: err.Details,
	})
}
