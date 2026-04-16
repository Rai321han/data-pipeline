package services

import (
	"errors"
	"fmt"
)

var (
	ErrValidation    = errors.New("validation error")
	ErrConfiguration = errors.New("configuration error")
	ErrStorage       = errors.New("storage error")
	ErrLLM           = errors.New("llm error")
	ErrProcessing    = errors.New("processing error")
	ErrSerialization = errors.New("serialization error")
)

type ServiceError struct {
	sentinel error
	Code     string
	Message  string
	Err      error
}

func NewServiceError(sentinel error, code, message string, err error) *ServiceError {
	return &ServiceError{
		sentinel: sentinel,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}

	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *ServiceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.sentinel
}
