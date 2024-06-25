package types

import (
	"fmt"
)

type ErrorResponse struct {
	ErrorType   string       `json:"error,omitempty"`
	Description string       `json:"description,omitempty"`
	BrokerError *BrokerError `json:"broker_error,omitempty"`
}

func (e *ErrorResponse) Error() string {
	if e.BrokerError != nil {
		return e.BrokerError.Error()
	}

	return e.Description
}

type BrokerError struct {
	StatusCode    int
	ErrorMessage  *string
	Description   *string
	ResponseError error
}

func (e *BrokerError) Error() string {
	var message, description string

	if e.ErrorMessage != nil {
		message = *e.ErrorMessage
	}
	if e.Description != nil {
		description = *e.Description
	}

	return fmt.Sprintf("BrokerError:%s, Status: %d, Description: %s", message, e.StatusCode, description)
}
