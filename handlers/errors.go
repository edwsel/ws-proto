package handlers

type EmptyEventError struct {
}

func NewEmptyEventError() *EmptyEventError {
	return &EmptyEventError{}
}

func (e *EmptyEventError) Error() string {
	return "empty event"
}
