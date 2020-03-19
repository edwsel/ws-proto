package transport

type WriteClosedError struct {
	message string
}

func NewWriteClosedError(message string) *WriteClosedError {
	return &WriteClosedError{message: message}
}

func (e *WriteClosedError) Error() string {
	return e.message
}

type PingPongError struct {
	message string
}

func NewPingPongError(message string) *PingPongError {
	return &PingPongError{message: message}
}

func (e *PingPongError) Error() string {
	return e.message
}