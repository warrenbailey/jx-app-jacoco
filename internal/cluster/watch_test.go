package cluster

import (
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateFact(t *testing.T) {
	report := report.Report{
		Counters: []report.Counter{
			{
				Type:    "INSTRUCTION",
				Missed:  10,
				Covered: 90,
			},
		},
	}

	url := "http://dummy"
	fact := createFact(report, url)

	assert.Equal(t, url, fact.Original.URL)
	assert.Equal(t, jenkinsv1.FactTypeCoverage, fact.FactType)
	assert.Len(t, fact.Measurements, 3)
	assert.Contains(t, fact.Measurements, jenkinsv1.Measurement{Name: "Instructions-Coverage", MeasurementType: "percent", MeasurementValue: 90})
	assert.Contains(t, fact.Measurements, jenkinsv1.Measurement{Name: "Instructions-Missed", MeasurementType: "percent", MeasurementValue: 10})
	assert.Contains(t, fact.Measurements, jenkinsv1.Measurement{Name: "Instructions-Total", MeasurementType: "percent", MeasurementValue: 100})
}
