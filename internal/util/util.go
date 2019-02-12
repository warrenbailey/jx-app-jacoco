package util

import (
	"github.com/cenkalti/backoff"
	"os"
	"time"
)

const (
	namespaceKey     = "TEAM_NAMESPACE"
	defaultNameSpace = "jx"
)

var timeout = 60 * time.Second

// TeamNameSpace returns the current namespace which is either defined by the TEAM_NAMESPACE environment variable or
// defaulted to 'jx'.
func TeamNameSpace() string {
	ns := os.Getenv(namespaceKey)
	if ns == "" {
		ns = defaultNameSpace
	}
	return ns
}

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
