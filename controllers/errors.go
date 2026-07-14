package controllers

import "github.com/kyma-project/btp-manager/internal/conditions"

type ErrorWithReason = conditions.ErrorWithReason

func NewErrorWithReason(reason conditions.Reason, message string) *ErrorWithReason {
	return conditions.NewErrorWithReason(reason, message)
}

type CertificateSignError struct {
	message string
}

func NewCertificateSignError(message string) CertificateSignError {
	return CertificateSignError{
		message: message,
	}
}

func (e CertificateSignError) Error() string {
	return e.message
}
