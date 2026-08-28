package api

import (
	"encoding/json"
	"fmt"
)

type APIError struct {
	Code    string         `json:"error"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Extra   map[string]any `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func ParseAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status}
	var raw map[string]any
	if json.Unmarshal(body, &raw) == nil {
		code, _ := raw["code"].(string)
		errField, _ := raw["error"].(string)
		message, _ := raw["message"].(string)
		switch {
		case code != "":
			e.Code = code
			e.Message = firstNonEmpty(message, errField)
		case errField != "":
			e.Code = errField
			e.Message = firstNonEmpty(message, errField)
		}
		delete(raw, "code")
		delete(raw, "error")
		delete(raw, "message")
		if len(raw) > 0 {
			e.Extra = raw
		}
	}
	if e.Code == "" {
		e.Code = "http_error"
	}
	if e.Message == "" {
		e.Message = fmt.Sprintf("HTTP %d", status)
	}
	return e
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
