package client

import "icoo_claw/common/httperr"

// HTTPError is a compatibility alias for Gateway internals. New shared callers
// should prefer httperr.HTTPError directly.
type HTTPError = httperr.HTTPError

func newHTTPError(service, method, path string, statusCode int, body []byte) *HTTPError {
	return httperr.New(service, method, path, statusCode, body)
}
