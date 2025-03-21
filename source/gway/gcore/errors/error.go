package errors

import "fmt"

type CodeError interface {
	error
	Code() int
}

func NewCodeError(code int, format string, args ...interface{}) CodeError {
	info := format
	if len(args) > 0 {
		info = fmt.Sprintf(format, args...)
	}
	return &codeError{
		code: code,
		info: info,
	}
}

type codeError struct {
	code int
	info string
}

func (c *codeError) Code() int {
	return c.code
}

func (c *codeError) Error() string {
	return c.info
}

func (c *codeError) String() string {
	return fmt.Sprintf("%d:%s", c.code, c.info)
}
