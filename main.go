package main

import (
	"crypto/tls"
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

func watch() (err error) {
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
	watch, err := client.PipelineActivities(ns).Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	var httpClient = &http.Client{
		Transport: tr,
		Timeout:   time.Second * 10,
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
						found := 0
						for i, f := range act.Spec.Facts {
							if f.FactType == jenkinsv1.FactTypeCoverage {
								act.Spec.Facts[i] = fact
								found++
							}
						}
						if found > 1 {
							return errors.New(fmt.Sprintf("More than one fact of kind %s found %d", jenkinsv1.FactTypeCoverage, found))
						} else if found == 0 {
							act.Spec.Facts = append(act.Spec.Facts, fact)
						}
						act, err = client.PipelineActivities(act.Namespace).Update(act)
						log.Printf("Updated PipelineActivity %s with data from %s\n", act.Name, url)
						if err != nil {
							log.Println(errors.Wrap(err, fmt.Sprintf("Error updating PipelineActivity %s", act.Name)))
						}
					}
				}
			}
		}
	}
	return nil
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
	go func() {
		err := watch()
		if err != nil {
			log.Fatal(err)
		}
	}()
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
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
