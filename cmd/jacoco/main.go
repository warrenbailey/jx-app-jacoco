package main

import (
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/cluster"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

var logger = log.WithFields(log.Fields{"app": "jacoco"})

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
}

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("unable to create in cluster config: %s", err)
	}

	jxClient, err := jenkinsclientv1.NewForConfig(config)
	if err != nil {
		logger.Fatalf("unable to create Jenkins client: %s", err)
	}

	ns := util.TeamNameSpace()
	logger.Infof("Watching namespace %s", ns)

	cluster.WatchPipelineActivity(ns, jxClient)

	// no-op HTTP handler for health and liveliness checks
	http.HandleFunc("/", handler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err.Error())
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
}
