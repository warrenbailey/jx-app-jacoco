package cluster

import (
	"fmt"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/config"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/logging"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jenkinsv1types "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	// SyncPeriod is the cache sync period.
	SyncPeriod = time.Minute * 10
	resource   = "pipelineactivities"
	appName    = "jacoco"
)

var (
	logger = logging.AppLogger().WithFields(log.Fields{"component": "event-handler"})
)

// EventHandler defines the callback functions for CRD changes
type EventHandler interface {
	// Add is called when a CRD is created.
	Add(new interface{})

	// Update is called when a CRD is updated.
	Update(old interface{}, new interface{})

	// Delete is called when a CRD is deleted.
	Delete(obj interface{})

	// Start starts the event handler passing it a done channel.
	Start(done chan struct{})
}

type defaultEventHandler struct {
	jxClient jenkinsv1client.Interface
	config   config.JXConfig
}

// NewEventHandler creates a new event handler using the JX REST client.
// A instance of defaultEventHandler handles syncing of a single CRD type specified via crdType.
func NewEventHandler(jxClient jenkinsv1client.Interface, config config.JXConfig) (EventHandler, error) {
	return &defaultEventHandler{jxClient: jxClient, config: config}, nil
}

func (h *defaultEventHandler) Add(obj interface{}) {
	h.onPipelineActivity(obj)
}

func (h *defaultEventHandler) Update(oldObj interface{}, newObj interface{}) {
	h.onPipelineActivity(newObj)
}

func (h *defaultEventHandler) Delete(obj interface{}) {

}

func (h *defaultEventHandler) Start(done chan struct{}) {
	listWatch := cache.NewListWatchFromClient(h.jxClient.JenkinsV1().RESTClient(),
		resource,
		h.config.Namespace(),
		fields.Everything())

	handlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			h.Add(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			h.Update(old, new)
		},
		DeleteFunc: func(obj interface{}) {
		},
	}
	_, controller := cache.NewInformer(listWatch, nil, SyncPeriod, handlerFuncs)

	controller.Run(done)
}

func (h *defaultEventHandler) onPipelineActivity(obj interface{}) {
	pipelineActivity, ok := obj.(*jenkinsv1.PipelineActivity)
	if !ok {
		logger.Warnf("unexpected type %T", obj)
		return
	}

	logger.Debugf("processing pipeline activity '%s'", pipelineActivity.Name)
	for _, attachment := range pipelineActivity.Spec.Attachments {
		if attachment.Name != appName {
			continue
		}

		factsInterface := h.jxClient.JenkinsV1().Facts(h.config.Namespace())
		for _, url := range attachment.URLs {
			logger.Debugf("processing report '%s'", url)
			//  append version string to report URL to avoid any caching issues when retrieving the report
			urlWithTimestamp := fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))
			report, err := report.RetrieveReport(h.config.Namespace(), urlWithTimestamp)
			if err != nil {
				logger.Errorf("unable to retrieve report from %s: %s", url, err)
				continue
			}

			fact := h.createFact(report, pipelineActivity, url)
			err = h.storeFact(fact, factsInterface)
			if err != nil {
				logger.Errorf("error storing Fact %s: %s", fact.Spec.Name, err)
			} else {
				logger.Infof("successfully stored JaCoCo fact '%s' for report from %s", fact.Spec.Name, fact.Spec.Original.URL)
			}
		}
	}
}

func (h *defaultEventHandler) storeFact(fact *jenkinsv1.Fact, factsInterface jenkinsv1types.FactInterface) error {
	f := func() error {
		_, err := factsInterface.Create(fact)
		if err != nil {
			switch err.(type) {
			case *errors.StatusError:
				status := err.(*errors.StatusError)
				if status.ErrStatus.Reason == metav1.StatusReasonAlreadyExists {
					logger.Debugf("fact with name '%s' already existed", fact.Name)
					return nil
				}
				return err
			default:
				return err
			}
		}
		return nil
	}
	return util.ApplyWithBackoff(f)
}

func (h *defaultEventHandler) createFact(report report.Report, pipelineActivity *jenkinsv1.PipelineActivity, url string) *jenkinsv1.Fact {
	measurements := make([]jenkinsv1.Measurement, 0)
	for _, c := range report.Counters {
		t := ""
		switch c.Type {
		case "INSTRUCTION":
			t = jenkinsv1.CodeCoverageCountTypeInstructions
		case "LINE":
			t = jenkinsv1.CodeCoverageCountTypeLines
		case "METHOD":
			t = jenkinsv1.CodeCoverageCountTypeMethods
		case "COMPLEXITY":
			t = jenkinsv1.CodeCoverageCountTypeComplexity
		case "BRANCH":
			t = jenkinsv1.CodeCoverageCountTypeBranches
		case "CLASS":
			t = jenkinsv1.CodeCoverageCountTypeClasses
		}
		measurementCovered := h.createMeasurement(t, jenkinsv1.CodeCoverageMeasurementCoverage, c.Covered)
		measurementMissed := h.createMeasurement(t, jenkinsv1.CodeCoverageMeasurementMissed, c.Missed)
		measurementTotal := h.createMeasurement(t, jenkinsv1.CodeCoverageMeasurementTotal, c.Covered+c.Missed)
		measurements = append(measurements, measurementCovered, measurementMissed, measurementTotal)
	}

	name := fmt.Sprintf("%s-%s-%s", appName, jenkinsv1.FactTypeCoverage, pipelineActivity.Name)
	fact := jenkinsv1.Fact{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: jenkinsv1.FactSpec{
			Name:     name,
			FactType: jenkinsv1.FactTypeCoverage,
			Original: jenkinsv1.Original{
				URL:      url,
				MimeType: "application/xml",
				Tags: []string{
					"jacoco.xml",
				},
			},
			Tags: []string{
				appName,
			},
			Measurements: measurements,
			Statements:   []jenkinsv1.Statement{},
			SubjectReference: jenkinsv1.ResourceReference{
				APIVersion: pipelineActivity.APIVersion,
				Kind:       pipelineActivity.Kind,
				Name:       pipelineActivity.Name,
				UID:        pipelineActivity.UID,
			},
		},
	}
	logger.Tracef("created fact: %v", fact)
	return &fact
}

func (h *defaultEventHandler) createMeasurement(t string, measurement string, value int) jenkinsv1.Measurement {
	return jenkinsv1.Measurement{
		Name:             fmt.Sprintf("%s-%s", t, measurement),
		MeasurementType:  jenkinsv1.MeasurementCount,
		MeasurementValue: value,
	}
}
