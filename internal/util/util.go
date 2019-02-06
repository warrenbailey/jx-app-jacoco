package util

import (
	"log"
	"os"
)

const namespaceKey = "TEAM_NAMESPACE"
const defaultNameSpace = "jx"

// TeamNameSpace returns the current namespace which is either defined by the TEAM_NAMESPACE environment variable or
// defaulted to 'jx'.
func TeamNameSpace() string {
	ns := os.Getenv(namespaceKey)
	if ns == "" {
		ns = defaultNameSpace
	}
	log.Printf("Using namespace %s", ns)
	return ns
}
