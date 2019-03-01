package cluster

import (
	"fmt"
	"github.com/bxcodec/faker"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
)

var dummyFact = &jenkinsv1.Fact{
	Spec: jenkinsv1.FactSpec{},
}

type mockFactInterface struct {
	createCount int
}

func (m *mockFactInterface) Facts(namespace string) jenkinsclientv1.FactInterface {
	return m
}

func (m *mockFactInterface) Create(*jenkinsv1.Fact) (*jenkinsv1.Fact, error) {
	m.createCount++

	if m.createCount < 3 {
		return nil, errors.New("dummy error")
	}
	return nil, nil
}

func (m *mockFactInterface) Update(*jenkinsv1.Fact) (*jenkinsv1.Fact, error) {
	return nil, nil
}

func (m *mockFactInterface) Delete(name string, options *meta_v1.DeleteOptions) error {
	return nil
}

func (m *mockFactInterface) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return nil
}

func (m *mockFactInterface) Get(name string, options meta_v1.GetOptions) (*jenkinsv1.Fact, error) {
	return nil, nil
}

func (m *mockFactInterface) List(opts meta_v1.ListOptions) (*jenkinsv1.FactList, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFactInterface) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFactInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *jenkinsv1.Fact, err error) {
	return nil, errors.New("not implemented")
}

func TestUpdatePipelineUpdateOperationIsRetried(t *testing.T) {
	mock := &mockFactInterface{}
	handler := defaultEventHandler{}
	handler.storeFact(dummyFact, mock)
	assert.Equal(t, 3, mock.createCount, "Expected get to be called 3 times")
}

func TestCreateFact(t *testing.T) {
	pipelineActivity := getFakePipelineActivity(t)
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

	handler := defaultEventHandler{}
	fact := handler.createFact(report, pipelineActivity, url)

	expectedName := fmt.Sprintf("%s-%s-%s", appName, jenkinsv1.FactTypeCoverage, pipelineActivity.Name)
	assert.Equal(t, expectedName, fact.Spec.Name)
	assert.Equal(t, url, fact.Spec.Original.URL)
	assert.Equal(t, jenkinsv1.FactTypeCoverage, fact.Spec.FactType)
	assert.Len(t, fact.Spec.Measurements, 3)
	assert.Contains(t, fact.Spec.Measurements, jenkinsv1.Measurement{Name: "Instructions-Covered", MeasurementType: jenkinsv1.MeasurementCount, MeasurementValue: 90})
	assert.Contains(t, fact.Spec.Measurements, jenkinsv1.Measurement{Name: "Instructions-Missed", MeasurementType: jenkinsv1.MeasurementCount, MeasurementValue: 10})
	assert.Contains(t, fact.Spec.Measurements, jenkinsv1.Measurement{Name: "Instructions-Total", MeasurementType: jenkinsv1.MeasurementCount, MeasurementValue: 100})
}

// GetFakePipelineActivity returns a PipelineActivity with fake data
func getFakePipelineActivity(t *testing.T) *jenkinsv1.PipelineActivity {
	activity := &jenkinsv1.PipelineActivity{}
	err := faker.FakeData(activity)
	if err != nil {
		t.Fatalf("Unable to mock CRDModel: %s", err)
	}
	return activity
}
