package generic

import (
	"errors"
	"fmt"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errInvalidMode   = errors.New("invalid datasource mode")
	errUnimplemented = errors.New("method implementation is missing on the server")

	ErrReqUnmarshalFailed = errors.New("req unmarshal failed")
)

type notFoundError string

func notFound(kind, name string) error {
	return notFoundError(fmt.Sprintf("%s not found: %s", kind, name))
}

func (e notFoundError) Error() string {
	return string(e)
}

func isNotFoundError(err error) bool {
	if grpcreflect.IsElementNotFoundError(err) {
		return true
	}

	_, ok := err.(notFoundError)
	return ok
}

func parseGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
		return errUnimplemented
	}

	return err
}
