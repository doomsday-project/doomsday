package doomsday

import "fmt"

type ErrUnauthorized struct {
	message string
}

func (e *ErrUnauthorized) Error() string {
	return e.message
}

type ErrBadRequest struct {
	message string
}

func (e *ErrBadRequest) Error() string {
	return e.message
}

type ErrInternalServer struct {
	message string
}

func (e *ErrInternalServer) Error() string {
	return e.message
}

func parseError(code int) (err error) {
	switch code {
	case 400:
		err = &ErrBadRequest{message: "400 - Bad Request"}
	case 401:
		err = &ErrUnauthorized{message: "401 - Unauthorized"}
	case 500:
		err = &ErrInternalServer{message: "500 - Internal Server Error"}
	default:
		err = fmt.Errorf("%d - An error occurred", code)
	}
	return err
}
