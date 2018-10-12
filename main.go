package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jenkins-x/ext-jacoco/jacoco"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const coverageFactName = "jenkins-x.coverage"

func watch() (err error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	ns := os.Getenv("TEAM_NAMESPACE")
	log.Printf("Using namespace %s", ns)
	client, err := jenkinsclientv1.NewForConfig(config)
	if err != nil {
		return err
	}
	watch, err := client.PipelineActivities(ns).Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	for event := range watch.ResultChan() {
		act, ok := event.Object.(*jenkinsv1.PipelineActivity)
		if !ok {
			log.Fatalf("unexpected type %s\n", event)
		}

		for _, attachment := range act.Spec.Attachments {
			if attachment.Name == "jacoco" {
				// TODO Handle having multiple attachments properly
				for _, url := range attachment.URLs {
					url = fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))
					report, err := parseReport(url, httpClient)
					if err != nil {
						log.Println(errors.Wrap(err, fmt.Sprintf("Unable to retrieve %s for processing", url)))
					}
					found := make([]jenkinsv1.Fact, 0)
					for _, f := range act.Spec.Facts {
						if f.FactType == coverageFactName {
							found = append(found, f)
							break
						}
					}
					if len(found) > 1 {
						return errors.New(fmt.Sprintf("More than one fact of kind %s found %s", coverageFactName, found))
					}
					fact := jenkinsv1.Fact{}
					if fact.Name == "" {
						fact.FactType = coverageFactName
						fact.Original = jenkinsv1.Original{
							URL:      url,
							MimeType: "application/xml",
							Tags: []string{
								"jacoco.xml",
							},
						}
						fact.Tags = []string{
							"jacoco",
						}
						act.Spec.Facts = append(act.Spec.Facts, fact)
					}
					measurements := make([]jenkinsv1.Measurement, 0)
					for _, c := range report.Counters {
						measurements = append(measurements, createMeasurement(c.Type, jenkinsv1.CodeCoverageMeasurementCoverage, c.Covered), createMeasurement(c.Type, jenkinsv1.CodeCoverageMeasurementMissed, c.Missed), createMeasurement(c.Type, jenkinsv1.CodeCoverageMeasurementTotal, c.Covered+c.Missed))
					}
					fact.Measurements = measurements
					act, err = client.PipelineActivities(act.Namespace).Update(act)
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

// Checks if all the elements of strings2 are present in strings1
func contains(strings1 []string, strings2 []string) bool {
	found := true
	for _, s2 := range strings2 {
		found2 := false
		for _, s1 := range strings1 {
			if s1 == s2 {
				found2 = true
				break
			}
		}
		if !found2 {
			found = false
			break
		}
	}
	return found
}

func parseReport(url string, httpClient *http.Client) (report jacoco.Report, err error) {
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
	err := watch()
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
	title := "Ready :-D"

	from := ""
	if r.URL != nil {
		from = r.URL.String()
	}
	if from != "/favicon.ico" {
		log.Printf("title: %s\n", title)
	}

	fmt.Fprintf(w, title+"\n")
}

func createMeasurement(t string, measurement string, value int) jenkinsv1.Measurement {
	return jenkinsv1.Measurement{
		Name:             fmt.Sprintf("%s-%s", t, measurement),
		MeasurementType:  "percent",
		MeasurementValue: value,
	}
}
