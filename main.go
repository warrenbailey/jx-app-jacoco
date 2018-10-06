package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jenkins-x/ext-jacoco-analyzer/jacoco"

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
	ns := os.Getenv("JACOCO_NAMESPACE")
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
		//
		if act.Spec.Summaries.CodeCoverageAnalysis.Original.URL == "" {
			for _, attachment := range act.Spec.Attachments {
				if attachment.Name == "jacoco" {
					// TODO Handle having multiple attachments properly
					for _, url := range attachment.URLs {
						url = fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond))
						report, err := parseReport(url, httpClient)
						if err != nil {
							log.Println(errors.Wrap(err, fmt.Sprintf("Unable to retrieve %s for processing", url)))
						}

						counts := make(map[string]jenkinsv1.CodeCoverageAnalysisCount)
						for _, c := range report.Counters {
							counts[c.Type] = jenkinsv1.CodeCoverageAnalysisCount{
								Coverage: c.Covered,
								Missed:   c.Missed,
								Total:    c.Covered + c.Missed,
							}
						}
						act.Spec.Summaries.CodeCoverageAnalysis = jenkinsv1.CodeCoverageAnalysis{
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
							Counts: counts,
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
	err := watch()
	if err != nil {
		panic(err.Error())
	}

}
