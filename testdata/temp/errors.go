package gorio

import "fmt"

type Error struct {
	Code    int
	Message string
	Op      string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s (code: %d)", e.Op, e.Message, e.Code)
}

func newError(op string, code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Op:      op,
	}
}
