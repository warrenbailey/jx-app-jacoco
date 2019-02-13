package main

import (
	"context"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/cluster"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	logger.Infof("watching namespace %s", ns)

	var wg sync.WaitGroup
	done := make(chan struct{})
	defer close(done)

	wg.Add(1)
	go func() {
		defer wg.Done()
		setupSignalChannel(done)
		return
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cluster.WatchPipelineActivity(done, ns, jxClient.RESTClient(), jxClient)
		logger.Info("cluster activity monitor has shut down")
		return
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startHTTPServer(done)
		logger.Info("HTTP server has shut down")
		return
	}()

	wg.Wait()
	logger.Info("jacoco has successfully shut down")
}

func startHTTPServer(done chan struct{}) {
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	})

	go func() {
		// returns ErrServerClosed on graceful close
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Errorf("ListenAndServe(): %s", err)
			done <- struct{}{}
		}
	}()

	select {
	case <-done:
		server.Shutdown(context.TODO())
	}

	return
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(done chan struct{}) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		logger.Info("waiting for shutdown signal in the background")
		<-sigChan
		logger.Info("received SIGTERM signal - initiating shutdown")
		done <- struct{}{}
	}()
}
