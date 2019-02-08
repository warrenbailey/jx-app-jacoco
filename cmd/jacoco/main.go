package main

import (
	"fmt"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	"net/http"
	"os"
	"time"

	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var logger = log.WithFields(log.Fields{"app": "jacoco"})

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
}

func main() {
	err := actWatch()
	if err != nil {
		logger.Fatal(err)
	}
	http.HandleFunc("/", handler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err.Error())
	}
}

func actWatch() (err error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	client, err := jenkinsclientv1.NewForConfig(config)
	if err != nil {
		return err
	}

	listWatch := cache.NewListWatchFromClient(client.RESTClient(), "pipelineactivities", util.TeamNameSpace(), fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, actController := cache.NewInformer(
		listWatch,
		&jenkinsv1.PipelineActivity{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				onPipelineActivityObj(obj, client)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				onPipelineActivityObj(newObj, client)
			},
			DeleteFunc: func(obj interface{}) {},
		},
	)
	stop := make(chan struct{})
	go actController.Run(stop)

	return nil
}

func onPipelineActivityObj(obj interface{}, jxClient *jenkinsclientv1.JenkinsV1Client) {
	act, ok := obj.(*jenkinsv1.PipelineActivity)
	if !ok {
		logger.Warnf("unexpected type %T", obj)
	} else {
		err := onPipelineActivity(act, jxClient)
		if err != nil {
			logger.Error(err)
		}
	}
}

func onPipelineActivity(act *jenkinsv1.PipelineActivity, jxClient *jenkinsclientv1.JenkinsV1Client) (err error) {
	for _, attachment := range act.Spec.Attachments {
		if attachment.Name == "jacoco" {
			// TODO Handle having multiple attachments properly
			for _, url := range attachment.URLs {
				url = fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))
				report, err := report.RetrieveReport(url)
				if err != nil {
					logger.Errorf("unable to retrieve %s for processing: %s", url, err)
				} else {
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
						measurements = append(measurements, createMeasurement(t, jenkinsv1.CodeCoverageMeasurementCoverage, c.Covered), createMeasurement(t, jenkinsv1.CodeCoverageMeasurementMissed, c.Missed), createMeasurement(t, jenkinsv1.CodeCoverageMeasurementTotal, c.Covered+c.Missed))
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
					newAct, err := jxClient.PipelineActivities(act.Namespace).Get(act.Name, metav1.GetOptions{})
					if err != nil {
						logger.Errorf("error updating PipelineActivity %s: %s", act.Name, err)
						continue
					}
					found := 0
					for i, f := range newAct.Spec.Facts {
						if f.FactType == jenkinsv1.FactTypeCoverage {
							newAct.Spec.Facts[i] = fact
							found++
						}
					}
					if found > 1 {
						return errors.New(fmt.Sprintf("more than one fact of kind %s found %d", jenkinsv1.FactTypeCoverage, found))
					} else if found == 0 {
						newAct.Spec.Facts = append(newAct.Spec.Facts, fact)
					}
					act, err = jxClient.PipelineActivities(newAct.Namespace).Update(newAct)
					logger.Infof("successfully updated PipelineActivity %s with data from %s", act.Name, url)
					if err != nil {
						logger.Errorf("error updating PipelineActivity %s: %s", act.Name, err)
					}
				}
			}
		}
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
}

func createMeasurement(t string, measurement string, value int) jenkinsv1.Measurement {
	return jenkinsv1.Measurement{
		Name:             fmt.Sprintf("%s-%s", t, measurement),
		MeasurementType:  "percent",
		MeasurementValue: value,
	}
}
