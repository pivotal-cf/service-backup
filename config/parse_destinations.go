package config

import (
	"code.cloudfoundry.org/lager"
	"fmt"

	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/backup"
	"github.com/pivotal-cf/service-backup/gcp"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
	"github.com/pivotal-cf/service-backup/systemtruststorelocator"
)

//go:generate counterfeiter -o configfakes/fake_system_trust_store_locator.go . SystemTrustStoreLocator
type SystemTrustStoreLocator interface {
	Path() (string, error)
}

//go:generate counterfeiter -o configfakes/fake_backuper_factory.go . BackuperFactory
type BackuperFactory interface {
	S3(destination Destination, locator SystemTrustStoreLocator) (*s3.S3CliClient, error)
	SCP(destination Destination) *scp.SCPClient
	Azure(destination Destination) *azure.AzureClient
	GCP(destination Destination) *gcp.StorageClient
}

func ParseDestinations(
	backupConfig BackupConfig,
	backuperFactory BackuperFactory,
	logger lager.Logger,
) ([]backup.Backuper, error) {

	var backupers []backup.Backuper

	for _, destination := range backupConfig.Destinations {
		switch destination.Type {
		case "s3":
			locator := systemtruststorelocator.New(RealFileSystem{})
			backuper, err := backuperFactory.S3(destination, locator)
			if err != nil {
				logger.Error("error configuring S3 destination", err)
				return nil, err
			}
			backupers = append(backupers, backuper)
		case "scp":
			backupers = append(backupers, backuperFactory.SCP(destination))
		case "azure":
			backupers = append(backupers, backuperFactory.Azure(destination))
		case "gcs":
			backupers = append(backupers, backuperFactory.GCP(destination))
		default:
			err := fmt.Errorf("unknown destination type: %s", destination.Type)
			logger.Error("error parsing destinations", err)
			return nil, err
		}
	}

	return backupers, nil
}
