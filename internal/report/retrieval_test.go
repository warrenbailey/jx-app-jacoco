package report

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

type errorThrowingRetriever struct {
}

func (r *errorThrowingRetriever) getRawReport(url string) ([]byte, error) {
	return nil, errors.New("Unable to retrieve report")
}

type fixtureRetriever struct {
}

func (r *fixtureRetriever) getRawReport(url string) ([]byte, error) {
	data, err := ioutil.ReadFile("testdata/jacoco.xml")
	return data, err
}

func TestRetrieveReportWithError(t *testing.T) {
	origRetriever := r
	defer func() {
		r = origRetriever
	}()
	r = &errorThrowingRetriever{}

	report, err := RetrieveReport("http://foo.bar/jacoco.xml")
	assert.Error(t, err)
	assert.Equal(t, "Unable to retrieve report", err.Error())
	assert.Equal(t, Report{}, report)
}

func TestRetrieveReportSuccess(t *testing.T) {
	origRetriever := r
	defer func() {
		r = origRetriever
	}()
	r = &fixtureRetriever{}

	report, err := RetrieveReport("http://foo.bar/jacoco.xml")
	assert.NoError(t, err)
	assert.Equal(t, "demo", report.Name)
}
