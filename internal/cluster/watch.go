package cluster

import (
	"fmt"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"time"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	logger = log.WithFields(log.Fields{"app": "jacoco"})
)

// WatchPipelineActivity watches the jx namespace for changes to PipelineActivities.
func WatchPipelineActivity(namespace string, jxClient *jenkinsclientv1.JenkinsV1Client) {
	listWatch := cache.NewListWatchFromClient(jxClient.RESTClient(), "pipelineactivities", namespace, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, actController := cache.NewInformer(
		listWatch,
		&jenkinsv1.PipelineActivity{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				onPipelineActivity(obj, jxClient)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				onPipelineActivity(newObj, jxClient)
			},
			DeleteFunc: func(obj interface{}) {},
		},
	)
	stop := make(chan struct{})
	go actController.Run(stop)
}

func onPipelineActivity(obj interface{}, jxClient *jenkinsclientv1.JenkinsV1Client) {
	pipelineActivity, ok := obj.(*jenkinsv1.PipelineActivity)
	if !ok {
		logger.Warnf("unexpected type %T", obj)
	} else {
		err := handlePipelineActivity(pipelineActivity, jxClient)
		if err != nil {
			logger.Error(err)
		}
	}
}

func handlePipelineActivity(pipelineActivity *jenkinsv1.PipelineActivity, jxClient *jenkinsclientv1.JenkinsV1Client) (err error) {
	for _, attachment := range pipelineActivity.Spec.Attachments {
		if attachment.Name != "jacoco" {
			continue
		}

		// TODO Handle having multiple attachments properly
		for _, url := range attachment.URLs {
			urlWithTimestamp := fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))

			if containsFactForURL(pipelineActivity, url) {
				continue
			}

			report, err := report.RetrieveReport(urlWithTimestamp)
			if err != nil {
				logger.Errorf("unable to retrieve %s for processing: %s", url, err)
				continue
			}

			fact := createFact(report, url)
			updatePipelineActivity(pipelineActivity.Name, pipelineActivity.Namespace, fact, jxClient)
		}
	}
	return nil
}

func containsFactForURL(pipelineActivity *jenkinsv1.PipelineActivity, url string) bool {
	for _, fact := range pipelineActivity.Spec.Facts {
		if fact.FactType == jenkinsv1.FactTypeCoverage && fact.Original.URL == url {
			return true
		}
	}
	return false
}

func updatePipelineActivity(name string, namespace string, fact jenkinsv1.Fact, jxClient *jenkinsclientv1.JenkinsV1Client) {
	var pipelineActivity *jenkinsv1.PipelineActivity
	var err error
	// re-get the pipeline activity
	f := func() error {
		pipelineActivity, err = jxClient.PipelineActivities(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	err = util.ApplyWithBackoff(f)
	if err != nil {
		logger.Errorf("error retrieving PipelineActivity %s with uuid %s for update: %s", pipelineActivity.Name, pipelineActivity.UID, err)
		return
	}

	found, index, err := indexOfJacocoCoverageFactType(pipelineActivity)
	if err != nil {
		logger.Error(err)
	}

	if found {
		// Updating the existing jacoco report with the new one. Is this what we want? (HF)
		pipelineActivity.Spec.Facts[index] = fact
	} else {
		pipelineActivity.Spec.Facts = append(pipelineActivity.Spec.Facts, fact)
	}

	// update the pipeline CRD
	f = func() error {
		_, err = jxClient.PipelineActivities(pipelineActivity.Namespace).Update(pipelineActivity)
		if err != nil {
			return err
		}
		return nil
	}
	err = util.ApplyWithBackoff(f)
	if err != nil {
		logger.Errorf("error updating PipelineActivity %s: %s", pipelineActivity.Name, err)
	} else {
		logger.Infof("successfully updated PipelineActivity %s with data from %s", pipelineActivity.Name, fact.Original.URL)
	}
}

func indexOfJacocoCoverageFactType(pipelineActivity *jenkinsv1.PipelineActivity) (bool, int, error) {
	found := 0
	index := -1
	for i, fact := range pipelineActivity.Spec.Facts {
		if fact.FactType == jenkinsv1.FactTypeCoverage && util.Contains(fact.Tags, "jacoco") {
			found++
			index = i
		}
	}
	if found > 1 {
		return false, index, errors.Errorf("multiple jacoco facts already attached to PipeLineActivity %s", pipelineActivity.Name)
	} else if found == 1 {
		return true, index, nil
	} else {
		return false, index, nil
	}
}

func createFact(report report.Report, url string) jenkinsv1.Fact {
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
		measurementCovered := createMeasurement(t, jenkinsv1.CodeCoverageMeasurementCoverage, c.Covered)
		measurementMissed := createMeasurement(t, jenkinsv1.CodeCoverageMeasurementMissed, c.Missed)
		measurementTotal := createMeasurement(t, jenkinsv1.CodeCoverageMeasurementTotal, c.Covered+c.Missed)
		measurements = append(measurements, measurementCovered, measurementMissed, measurementTotal)
	}
	fact := jenkinsv1.Fact{
		FactType: jenkinsv1.FactTypeCoverage,
		Original: jenkinsv1.Original{
			URL:      url,
			MimeType: "application/xml",
			Tags: []string{
				"jacoco.xml",
			},
		},
		Tags: []string{
			"jacoco",
		},
		Measurements: measurements,
		Statements:   []jenkinsv1.Statement{},
	}
	return fact
}

func createMeasurement(t string, measurement string, value int) jenkinsv1.Measurement {
	return jenkinsv1.Measurement{
		Name:             fmt.Sprintf("%s-%s", t, measurement),
		MeasurementType:  "percent",
		MeasurementValue: value,
	}
}
