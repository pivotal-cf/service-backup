package upload

import (
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/gcp"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
)

type Uploader interface {
	Upload(localPath string, sessionLogger lager.Logger) error
	Name() string
}

//go:generate counterfeiter -o fakes/uploader_factory.go . UploaderFactory
type UploaderFactory interface {
	S3(destination config.Destination, caCertPath string) *s3.S3CliClient
	SCP(destination config.Destination) *scp.SCPClient
	Azure(destination config.Destination) *azure.AzureClient
	GCP(destination config.Destination) *gcp.StorageClient
}

func Initialize(conf *config.BackupConfig, logger lager.Logger, options ...Option) (*multiUploader, error) {
	opts := &opts{
		factory:       &uploaderFactory{backupConfig: conf},
		caCertLocator: CACertPath,
	}

	for _, opt := range options {
		opt(opts)
	}

	uploaders := make([]Uploader, len(conf.Destinations))

	for i, dest := range conf.Destinations {
		switch dest.Type {
		case "s3":
			caCert, err := opts.caCertLocator()
			if err != nil {
				return nil, err
			}
			uploaders[i] = opts.factory.S3(dest, caCert)
		case "scp":
			// TODO: add test for this branch
			uploaders[i] = opts.factory.SCP(dest)
		case "azure":
			// TODO: add test for this branch
			uploaders[i] = opts.factory.Azure(dest)
		case "gcs":
			// TODO: add test for this branch
			uploaders[i] = opts.factory.GCP(dest)
		default:
			err := fmt.Errorf("unknown destination type: %s", dest.Type)
			logger.Error("error parsing destinations", err)
			return nil, err
		}
	}

	return &multiUploader{uploaders}, nil
}
