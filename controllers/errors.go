package controllers

import "github.com/kyma-project/btp-manager/internal/conditions"

type ErrorWithReason struct {
	message string
	reason  conditions.Reason
}

func NewErrorWithReason(reason conditions.Reason, message string) *ErrorWithReason {
	return &ErrorWithReason{
		message: message,
		reason:  reason,
	}
}

func (e *ErrorWithReason) Error() string {
	return e.message
}
