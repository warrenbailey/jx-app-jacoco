package main

import (
	"context"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/cluster"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/config"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/logging"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	logger = logging.AppLogger().WithFields(log.Fields{"component": "main"})
)

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
}

func main() {
	// Init configuration
	config, err := config.NewConfiguration()
	if err != nil {
		logger.Fatal(err)
	}
	logger.Infof("starting %s with config: %s", logging.AppName, config)

	// configure the Logger
	logging.SetLevel(config.Level())

	factory := clients.NewFactory()
	jxClient, _, err := factory.CreateJXClient()
	if err != nil {
		logger.Fatal(err)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		setupSignalChannel(done)
		return
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		eventHandler, err := cluster.NewEventHandler(jxClient, config)
		if err != nil {
			logger.Errorf("error creating event handler: %s", err)
			done <- struct{}{}
			return
		}
		logger.Info("starting event handler for pipelineactivites")
		eventHandler.Start(done)

		logger.Info("event handler has shut down")
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
		close(done)
	}()
}
