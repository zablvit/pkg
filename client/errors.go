package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// IsNotFound returns true if the error represents a NotFound response from an
// upstream service.
func IsNotFound(err error) bool {
	e, ok := err.(SCMError)
	return ok && e.Status == http.StatusNotFound
}

type SCMError struct {
	Msg         string
	Status      int
	ResponseMsg string
}

func (s SCMError) Error() string {
	return fmt.Sprintf("%s: response status %d: %s", s.Msg, s.Status, s.ResponseMsg)
}

func newSCMError(msg string, status int, body io.ReadCloser) SCMError {
	defer func() { _ = body.Close() }()

	e := SCMError{
		Msg:    msg,
		Status: status,
	}

	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return e
	}

	e.ResponseMsg = string(bytes)
	return e
}
