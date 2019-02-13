package cluster

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
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
func WatchPipelineActivity(done chan struct{}, namespace string, restClient rest.Interface, pipelineActivitiesGetter jenkinsclientv1.PipelineActivitiesGetter) {
	listWatch := cache.NewListWatchFromClient(restClient, "pipelineactivities", namespace, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, actController := cache.NewInformer(
		listWatch,
		&jenkinsv1.PipelineActivity{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				onPipelineActivity(obj, pipelineActivitiesGetter)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				onPipelineActivity(newObj, pipelineActivitiesGetter)
			},
			DeleteFunc: func(obj interface{}) {},
		},
	)
	actController.Run(done)
}

func onPipelineActivity(obj interface{}, pipelineActivitiesGetter jenkinsclientv1.PipelineActivitiesGetter) {
	pipelineActivity, ok := obj.(*jenkinsv1.PipelineActivity)
	if !ok {
		logger.Warnf("unexpected type %T", obj)
	} else {
		err := handlePipelineActivity(pipelineActivity, pipelineActivitiesGetter)
		if err != nil {
			logger.Error(err)
		}
	}
}

func handlePipelineActivity(pipelineActivity *jenkinsv1.PipelineActivity, pipelineActivitiesGetter jenkinsclientv1.PipelineActivitiesGetter) (err error) {
	for _, attachment := range pipelineActivity.Spec.Attachments {
		if attachment.Name != "jacoco" {
			continue
		}

		for _, url := range attachment.URLs {
			//  append version string to report URL to avoid any caching issues when retrieving the report
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

			// retry fetch+update pipeline CRD as a whole

			err = updatePipelineActivity(pipelineActivity.Name, pipelineActivity.Namespace, fact, pipelineActivitiesGetter)
			if err != nil {
				logger.Errorf("error updating PipelineActivity %s: %s", pipelineActivity.Name, err)
			} else {
				logger.Infof("successfully updated PipelineActivity %s with data from %s", pipelineActivity.Name, fact.Original.URL)
			}
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

func updatePipelineActivity(name string, namespace string, fact jenkinsv1.Fact, pipelineActivitiesGetter jenkinsclientv1.PipelineActivitiesGetter) error {
	f := func() error {
		pipelineActivity, err := pipelineActivitiesGetter.PipelineActivities(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		found, index, err := indexOfJacocoCoverageFactType(pipelineActivity)
		if err != nil {
			return backoff.Permanent(err)
		}

		if found {
			pipelineActivity.Spec.Facts[index] = fact
		} else {
			pipelineActivity.Spec.Facts = append(pipelineActivity.Spec.Facts, fact)
		}

		_, err = pipelineActivitiesGetter.PipelineActivities(pipelineActivity.Namespace).Update(pipelineActivity)
		if err != nil {
			return err
		}
		return nil
	}
	return util.ApplyWithBackoff(f)
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
