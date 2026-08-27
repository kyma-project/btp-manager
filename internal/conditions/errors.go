package conditions

type ErrorWithReason struct {
	Message string
	Reason  Reason
}

func NewErrorWithReason(reason Reason, message string) *ErrorWithReason {
	return &ErrorWithReason{
		Message: message,
		Reason:  reason,
	}
}

func (e *ErrorWithReason) Error() string {
	return e.Message
}
