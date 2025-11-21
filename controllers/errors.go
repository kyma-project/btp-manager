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
