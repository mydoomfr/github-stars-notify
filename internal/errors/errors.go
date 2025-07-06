package errors

import (
	"errors"
	"fmt"
)

// Error types for different components
var (
	ErrConfiguration = errors.New("configuration error")
	ErrGitHubAPI     = errors.New("github api error")
	ErrStorage       = errors.New("storage error")
	ErrNotification  = errors.New("notification error")
	ErrService       = errors.New("service error")
	ErrValidation    = errors.New("validation error")
)

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ConfigurationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("configuration error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("configuration error: %s", e.Message)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

func (e *ConfigurationError) Is(target error) bool {
	return target == ErrConfiguration
}

// GitHubAPIError represents GitHub API-related errors
type GitHubAPIError struct {
	Endpoint   string
	StatusCode int
	Message    string
	Err        error
}

func (e *GitHubAPIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("github api error [%d] on %s: %s", e.StatusCode, e.Endpoint, e.Message)
	}
	return fmt.Sprintf("github api error on %s: %s", e.Endpoint, e.Message)
}

func (e *GitHubAPIError) Unwrap() error {
	return e.Err
}

func (e *GitHubAPIError) Is(target error) bool {
	return target == ErrGitHubAPI
}

// IsRateLimited checks if the error is due to rate limiting
func (e *GitHubAPIError) IsRateLimited() bool {
	return e.StatusCode == 403 || e.StatusCode == 429
}

// StorageError represents storage-related errors
type StorageError struct {
	Operation string
	Path      string
	Message   string
	Err       error
}

func (e *StorageError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("storage error during %s on %s: %s", e.Operation, e.Path, e.Message)
	}
	return fmt.Sprintf("storage error during %s: %s", e.Operation, e.Message)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

func (e *StorageError) Is(target error) bool {
	return target == ErrStorage
}

// NotificationError represents notification-related errors
type NotificationError struct {
	Provider string
	Message  string
	Err      error
}

func (e *NotificationError) Error() string {
	return fmt.Sprintf("notification error (%s): %s", e.Provider, e.Message)
}

func (e *NotificationError) Unwrap() error {
	return e.Err
}

func (e *NotificationError) Is(target error) bool {
	return target == ErrNotification
}

// ServiceError represents service-level errors
type ServiceError struct {
	Component string
	Message   string
	Err       error
}

func (e *ServiceError) Error() string {
	if e.Component != "" {
		return fmt.Sprintf("service error in %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("service error: %s", e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

func (e *ServiceError) Is(target error) bool {
	return target == ErrService
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// Helper functions for creating specific errors

// NewConfigurationError creates a new configuration error
func NewConfigurationError(field, message string, err error) *ConfigurationError {
	return &ConfigurationError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// NewGitHubAPIError creates a new GitHub API error
func NewGitHubAPIError(endpoint string, statusCode int, message string, err error) *GitHubAPIError {
	return &GitHubAPIError{
		Endpoint:   endpoint,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// NewStorageError creates a new storage error
func NewStorageError(operation, path, message string, err error) *StorageError {
	return &StorageError{
		Operation: operation,
		Path:      path,
		Message:   message,
		Err:       err,
	}
}

// NewNotificationError creates a new notification error
func NewNotificationError(provider, message string, err error) *NotificationError {
	return &NotificationError{
		Provider: provider,
		Message:  message,
		Err:      err,
	}
}

// NewServiceError creates a new service error
func NewServiceError(component, message string, err error) *ServiceError {
	return &ServiceError{
		Component: component,
		Message:   message,
		Err:       err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string, err error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Err:     err,
	}
}
