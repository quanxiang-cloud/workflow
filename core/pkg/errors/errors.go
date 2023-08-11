package errors

import (
	"encoding/json"
	"errors"
)

type Error struct {
	Code    int
	Message ErrorBody
}

func (e *Error) Error() string {
	if e.Message == nil {
		return ""
	}
	return e.Message.JSON()
}

// NewErr return error, rrror body only 0 or 1.
func NewErr(code int, eb ...ErrorBody) error {
	if len(eb) == 0 {
		return &Error{
			Code: code,
		}
	}

	return &Error{
		Code:    code,
		Message: eb[0],
	}
}

type ErrorBody interface {
	JSON() string
}

type CodeError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *CodeError) JSON() string {
	byte, _ := json.Marshal(c)
	return string(byte)
}

type StringError string

func (s *StringError) JSON() string {
	return "{\"message\":\"" + string(*s) + "\"}"
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
