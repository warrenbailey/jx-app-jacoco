package report

import (
	"encoding/xml"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"time"
)

var (
	timeout           = time.Second * 30
	r       retriever = &defaultRetriever{}
)

type retriever interface {
	getRawReport(namespace string, url string) ([]byte, error)
}

type defaultRetriever struct {
}

func (r *defaultRetriever) getRawReport(namepace string, url string) ([]byte, error) {
	common := cmd.NewCommonOptions(namepace, clients.NewFactory())

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

// RetrieveReport retrieves a JaCoCo report from the specified URL which can be on GitHub or a cloud storage bucket.
func RetrieveReport(namespace string, url string) (Report, error) {
	rawReport, err := r.getRawReport(namespace, url)
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
