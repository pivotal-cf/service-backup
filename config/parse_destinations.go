package config

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/gcp"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
)

//go:generate counterfeiter -o configfakes/fake_system_trust_store_locator.go . SystemTrustStoreLocator
type SystemTrustStoreLocator interface {
	Path() (string, error)
}

func ParseDestinations(backupConfig BackupConfig, systemTrustStoreLocator SystemTrustStoreLocator, logger lager.Logger) ([]backup.Backuper, error) {
	var backupers []backup.Backuper

	for _, destination := range backupConfig.Destinations {
		destinationConfig := destination.Config
		switch destination.Type {
		case "s3":
			basePath := fmt.Sprintf("%s/%s", destinationConfig["bucket_name"], destinationConfig["bucket_path"])
			systemTrustStorePath, err := systemTrustStoreLocator.Path()
			if err != nil {
				logger.Error("error locating system trust store for S3", err)
				return nil, err
			}
			backupers = append(backupers, s3.New(
				destination.Name,
				backupConfig.AwsCliPath,
				destinationConfig["endpoint_url"].(string),
				destinationConfig["access_key_id"].(string),
				destinationConfig["secret_access_key"].(string),
				basePath,
				systemTrustStorePath,
			))
		case "scp":
			basePath := destinationConfig["destination"].(string)
			backupers = append(backupers, scp.New(
				destination.Name,
				destinationConfig["server"].(string),
				destinationConfig["port"].(int),
				destinationConfig["user"].(string),
				destinationConfig["key"].(string),
				basePath,
				destinationConfig["fingerprint"].(string),
			))
		case "azure":
			basePath := destinationConfig["path"].(string)
			backupers = append(backupers, azure.New(
				destination.Name,
				destinationConfig["storage_access_key"].(string),
				destinationConfig["storage_account"].(string),
				destinationConfig["container"].(string),
				destinationConfig["blob_store_base_url"].(string),
				backupConfig.AzureCliPath,
				basePath,
			))
		case "gcs":
			backupers = append(backupers, gcp.New(
				destination.Name,
				os.Getenv("GCP_SERVICE_ACCOUNT_FILE"),
				destinationConfig["project_id"].(string),
				destinationConfig["bucket_name"].(string),
			))
		default:
			err := fmt.Errorf("unknown destination type: %s", destination.Type)
			logger.Error("error parsing destinations", err)
			return nil, err
		}
	}

	return backupers, nil
}
