package util

import (
	"github.com/cenkalti/backoff"
	"time"
)

var timeout = 60 * time.Second

// Contains checks whether the specified string is contained in the given string slice.
// Returns true if it does, false otherwise
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// ApplyWithBackoff tries to apply the specified function using an exponential backoff algorithm.
// If the function eventually succeed nil is returned, otherwise the error returned by f.
func ApplyWithBackoff(f func() error) error {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = timeout
	exponentialBackOff.Reset()
	return backoff.Retry(f, exponentialBackOff)
}
