package util

import (
	log "github.com/sirupsen/logrus"
	"os"
)

var logger = log.WithFields(log.Fields{"app": "jacoco"})

const namespaceKey = "TEAM_NAMESPACE"
const defaultNameSpace = "jx"

// TeamNameSpace returns the current namespace which is either defined by the TEAM_NAMESPACE environment variable or
// defaulted to 'jx'.
func TeamNameSpace() string {
	ns := os.Getenv(namespaceKey)
	if ns == "" {
		ns = defaultNameSpace
	}
	logger.Infof("Using namespace %s", ns)
	return ns
}
