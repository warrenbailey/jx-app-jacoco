package util

import (
	"github.com/magiconair/properties/assert"
	"os"
	"testing"
)

func TestTeamNameSpaceDefault(t *testing.T) {
	origNameSpace := os.Getenv(namespaceKey)
	defer func() {
		os.Setenv(namespaceKey, origNameSpace)
	}()

	os.Unsetenv(namespaceKey)
	ns := TeamNameSpace()
	assert.Equal(t, defaultNameSpace, ns)
}

func TestTeamNameSpaceExplicit(t *testing.T) {
	origNameSpace := os.Getenv(namespaceKey)
	defer func() {
		os.Setenv(namespaceKey, origNameSpace)
	}()

	os.Setenv(namespaceKey, "foo")
	ns := TeamNameSpace()
	assert.Equal(t, "foo", ns)
}
