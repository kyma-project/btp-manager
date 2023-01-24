package controllers

type ErrorWithReason struct {
	message string
	reason  Reason
}

func NewErrorWithReason(reason Reason, message string) *ErrorWithReason {
	return &ErrorWithReason{
		message: message,
		reason:  reason,
	}
}

func (e *ErrorWithReason) Error() string {
	return e.message
}
