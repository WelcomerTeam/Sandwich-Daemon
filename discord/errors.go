package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
)

var (
	ErrUnauthorized         = errors.New("improper token was passed")
	ErrUnsupportedImageType = errors.New("unsupported image type given")
)

// RestError contains the error structure that is returned by discord.
type RestError struct {
	Request      *http.Request
	Response     *http.Response
	Message      *ErrorMessage
	ResponseBody []byte
}

// ErrorMessage represents a basic error message.
type ErrorMessage struct {
	Message string          `json:"message"`
	Errors  json.RawMessage `json:"errors"`
	Code    int32           `json:"code"`
}

func NewRestError(req *http.Request, resp *http.Response, body []byte) *RestError {
	var errorMessage ErrorMessage

	_ = sandwichjson.Unmarshal(body, errorMessage)

	return &RestError{
		Request:      req,
		Response:     resp,
		ResponseBody: body,
		Message:      &errorMessage,
	}
}

func (r *RestError) Error() string {
	return fmt.Sprintf("%s: %s", r.Response.Status, r.Message.Message)
}
