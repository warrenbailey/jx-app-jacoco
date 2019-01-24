package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jenkins-x-apps/jx-app-jacoco/jacoco"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Retry policy retrying on 404
func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return true, err
	}
	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || resp.StatusCode == 404 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	return false, nil
}

func actWatch() (err error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	ns := os.Getenv("TEAM_NAMESPACE")
	if ns == "" {
		ns = "jx"
	}
	log.Printf("Using namespace %s", ns)
	client, err := jenkinsclientv1.NewForConfig(config)
	if err != nil {
		return err
	}

	var httpClient = retryablehttp.NewClient()
	httpClient.RetryWaitMin = 2 * time.Second
	httpClient.RetryMax = 5
	httpClient.CheckRetry = retryPolicy

	listWatch := cache.NewListWatchFromClient(client.RESTClient(), "pipelineactivities", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, actController := cache.NewInformer(
		listWatch,
		&jenkinsv1.PipelineActivity{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				onPipelineActivityObj(obj, httpClient, client)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				onPipelineActivityObj(newObj, httpClient, client)
			},
			DeleteFunc: func(obj interface{}) {

			},
		},
	)
	stop := make(chan struct{})
	go actController.Run(stop)

	return nil
}

func onPipelineActivityObj(obj interface{}, httpClient *retryablehttp.Client, jxClient *jenkinsclientv1.JenkinsV1Client) {
	act, ok := obj.(*jenkinsv1.PipelineActivity)
	if !ok {
		log.Printf("unexpected type %s\n", obj)
	} else {
		err := onPipelineActivity(act, httpClient, jxClient)
		if err != nil {
			log.Print(err)
		}
	}
}

func onPipelineActivity(act *jenkinsv1.PipelineActivity, httpClient *retryablehttp.Client, jxClient *jenkinsclientv1.JenkinsV1Client) (err error) {
	for _, attachment := range act.Spec.Attachments {
		if attachment.Name == "jacoco" {
			// TODO Handle having multiple attachments properly
			for _, url := range attachment.URLs {
				url = fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))
				report, err := parseReport(url, httpClient)
				if err != nil {
					log.Println(errors.Wrap(err, fmt.Sprintf("Unable to retrieve %s for processing", url)))
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
					}
					newAct, err := jxClient.PipelineActivities(act.Namespace).Get(act.Name, metav1.GetOptions{})
					if err != nil {
						log.Println(errors.Wrap(err, fmt.Sprintf("Error updating PipelineActivity %s", act.Name)))
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
						return errors.New(fmt.Sprintf("More than one fact of kind %s found %d", jenkinsv1.FactTypeCoverage, found))
					} else if found == 0 {
						newAct.Spec.Facts = append(newAct.Spec.Facts, fact)
					}
					act, err = jxClient.PipelineActivities(newAct.Namespace).Update(newAct)
					log.Printf("Updated PipelineActivity %s with data from %s\n", act.Name, url)
					if err != nil {
						log.Println(errors.Wrap(err, fmt.Sprintf("Error updating PipelineActivity %s", act.Name)))
					}
				}
			}
		}
	}
	return nil
}

func parseReport(url string, httpClient *retryablehttp.Client) (report jacoco.Report, err error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return jacoco.Report{}, err
	}
	if response.StatusCode > 299 || response.StatusCode < 200 {
		return jacoco.Report{}, errors.New(fmt.Sprintf("Status code: %d, error: %s", response.StatusCode, response.Status))
	}
	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return jacoco.Report{}, err
	}
	err = xml.Unmarshal(body, &report)
	if err != nil {
		return report, err
	}
	return report, nil
}

func main() {
	err := actWatch()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err.Error())
	}
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
