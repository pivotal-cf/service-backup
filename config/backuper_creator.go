package config

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/gcp"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
)

type BackuperCreator struct {
	backupConfig *BackupConfig
}

func NewBackuperCreator(bc *BackupConfig) *BackuperCreator {
	return &BackuperCreator{
		backupConfig: bc,
	}
}

func (b *BackuperCreator) S3(destination Destination, locator SystemTrustStoreLocator) (*s3.S3CliClient, error) {
	basePath := fmt.Sprintf(
		"%s/%s",
		destination.Config.getString("bucket_name"),
		destination.Config.getString("bucket_path"),
	)

	systemTrustStorePath, err := locator.Path()
	if err != nil {
		return nil, err
	}

	return s3.New(
		destination.Name,
		b.backupConfig.AwsCliPath,
		destination.Config.getString("endpoint_url"),
		destination.Config.getString("region"),
		destination.Config.getString("access_key_id"),
		destination.Config.getString("secret_access_key"),
		systemTrustStorePath,
		RemotePathGenerator{
			BasePath:       basePath,
			DeploymentName: b.backupConfig.DeploymentName,
		},
	), nil
}

func (b *BackuperCreator) SCP(destination Destination) *scp.SCPClient {
	return scp.New(
		destination.Name,
		destination.Config.getString("server"),
		destination.Config.getInt("port"),
		destination.Config.getString("user"),
		destination.Config.getString("key"),
		destination.Config.getString("fingerprint"),
		RemotePathGenerator{
			BasePath: destination.Config.getString("destination"),
		},
	)
}

func (b *BackuperCreator) Azure(destination Destination) *azure.AzureClient {
	return azure.New(
		destination.Name,
		destination.Config.getString("storage_access_key"),
		destination.Config.getString("storage_account"),
		destination.Config.getString("container"),
		destination.Config.getString("blob_store_base_url"),
		b.backupConfig.AzureCliPath,
		RemotePathGenerator{
			BasePath: destination.Config.getString("path"),
		},
	)
}

func (b *BackuperCreator) GCP(destination Destination) *gcp.StorageClient {
	return gcp.New(
		destination.Name,
		os.Getenv("GCP_SERVICE_ACCOUNT_FILE"),
		destination.Config.getString("project_id"),
		destination.Config.getString("bucket_name"),
		RemotePathGenerator{},
	)
}
