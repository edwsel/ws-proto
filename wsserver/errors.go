package wsserver

import "fmt"

type RaiseFailUsageError struct {
	message string
}

func NewRaiseFailUsageError(message string) *RaiseFailUsageError {
	return &RaiseFailUsageError{message: message}
}

func (e *RaiseFailUsageError) Error() string {
	return fmt.Sprintf("raise faile usege: %s", e.message)
}
