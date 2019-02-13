package cluster

import (
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/report"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsclientv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
)

var dummyPipeLineActivity = &jenkinsv1.PipelineActivity{
	Spec: jenkinsv1.PipelineActivitySpec{
		Facts: []jenkinsv1.Fact{},
	},
}

type mockPipelineActivitiesGetter struct {
	updateCount int
	getCount    int
}

func (m *mockPipelineActivitiesGetter) PipelineActivities(namespace string) jenkinsclientv1.PipelineActivityInterface {
	return m
}

func (m *mockPipelineActivitiesGetter) Create(*v1.PipelineActivity) (*v1.PipelineActivity, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPipelineActivitiesGetter) Update(*v1.PipelineActivity) (*v1.PipelineActivity, error) {
	m.updateCount++

	if m.updateCount < 3 {
		return nil, errors.New("dummy error")
	}
	return nil, nil
}

func (m *mockPipelineActivitiesGetter) Delete(name string, options *meta_v1.DeleteOptions) error {
	return nil
}

func (m *mockPipelineActivitiesGetter) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return nil
}

func (m *mockPipelineActivitiesGetter) Get(name string, options meta_v1.GetOptions) (*v1.PipelineActivity, error) {
	m.getCount++
	return dummyPipeLineActivity, nil
}

func (m *mockPipelineActivitiesGetter) List(opts meta_v1.ListOptions) (*v1.PipelineActivityList, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPipelineActivitiesGetter) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPipelineActivitiesGetter) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PipelineActivity, err error) {
	return nil, errors.New("not implemented")
}

func TestUpdatePipelineUpdateOperationIsRetried(t *testing.T) {
	mock := &mockPipelineActivitiesGetter{}
	dummyFact := jenkinsv1.Fact{
		Name: "dummy-fact",
	}
	updatePipelineActivity("pipeline-x", "jx", dummyFact, mock)
	assert.Equal(t, 3, mock.getCount, "Expected get to be called 3 times")
	assert.Equal(t, 3, mock.updateCount, "Expected update to be called 3 times")
}

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
