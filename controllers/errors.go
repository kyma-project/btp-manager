package controllers

import "github.com/kyma-project/btp-manager/internal/conditions"

type ErrorWithReason = conditions.ErrorWithReason

func NewErrorWithReason(reason conditions.Reason, message string) *ErrorWithReason {
	return conditions.NewErrorWithReason(reason, message)
}
