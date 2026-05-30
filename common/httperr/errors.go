package httperr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type HTTPError struct {
	Service    string
	Method     string
	Path       string
	StatusCode int
	Code       string
	Message    string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("%s %s %s: status %d: %s: %s", e.Service, e.Method, e.Path, e.StatusCode, e.Code, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s %s %s: status %d: %s", e.Service, e.Method, e.Path, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s %s %s: status %d", e.Service, e.Method, e.Path, e.StatusCode)
}

func New(service, method, path string, statusCode int, body []byte) *HTTPError {
	bodyText := strings.TrimSpace(string(body))
	err := &HTTPError{
		Service:    service,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Message:    bodyText,
	}

	var payload struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil {
		err.Code = payload.Code
		if payload.Error != "" {
			err.Message = payload.Error
		}
	}
	if err.Code == "" {
		err.Code = CodeForStatus(statusCode)
	}
	return err
}

func CodeForStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusRequestEntityTooLarge:
		return "payload_too_large"
	default:
		if statusCode >= 500 {
			return "dependency_unavailable"
		}
		return "downstream_error"
	}
}
