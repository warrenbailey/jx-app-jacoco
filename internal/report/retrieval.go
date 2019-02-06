package report

import (
	"encoding/xml"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"time"
)

var (
	timeout           = time.Second * 30
	r       retriever = &defaultRetriever{}
)

type retriever interface {
	getRawReport(url string) ([]byte, error)
}

type defaultRetriever struct {
}

func (r *defaultRetriever) getRawReport(url string) ([]byte, error) {
	common := cmd.NewCommonOptions(util.TeamNameSpace(), cmd.NewFactory())

	authSvc, err := common.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}

	data, err := buckets.ReadURL(url, timeout, cmd.CreateBucketHTTPFn(authSvc))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// RetrieveReport retrieves a Jacoco report from the specified URL which can be on GitHub or a cloud storage bucket.
func RetrieveReport(url string) (Report, error) {
	rawReport, err := r.getRawReport(url)
	if err != nil {
		return Report{}, err
	}
	report := Report{}
	err = xml.Unmarshal(rawReport, &report)
	if err != nil {
		return report, err
	}
	return report, nil
}
