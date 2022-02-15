package client

import (
	"fmt"
	"net/http"
)

// IsNotFound returns true if the error represents a NotFound response from an
// upstream service.
func IsNotFound(err error) bool {
	e, ok := err.(SCMError)
	return ok && e.Status == http.StatusNotFound
}

type SCMError struct {
	Msg    string
	Status int
}

func (s SCMError) Error() string {
	return fmt.Sprintf("%s: (%d)", s.Msg, s.Status)
}
