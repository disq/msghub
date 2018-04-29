package msghub

import "strings"

// IsErrNetClosing checks if the error is ErrNetClosing (defined in stdlib internal/poll/fd.go)
func IsErrNetClosing(err error) bool {
	// No other way to check
	return strings.Contains(err.Error(), "use of closed network connection")
}

// IsTimeoutError checks if the error is TimeoutError (defined in stdlib internal/poll/fd.go)
func IsTimeoutError(err error) bool {
	// No other way to check
	return strings.Contains(err.Error(), "i/o timeout")
}
