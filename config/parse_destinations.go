package config

import (
	"fmt"
	"os"

	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/gcp"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
)

func ParseDestinations(backupConfig BackupConfig) []backup.Backuper {
	var backupers []backup.Backuper

	for _, destination := range backupConfig.Destinations {
		destinationConfig := destination.Config
		switch destination.DestType {
		case "s3":
			basePath := fmt.Sprintf("%s/%s", destinationConfig["bucket_name"], destinationConfig["bucket_path"])
			backupers = append(backupers, s3.New(
				destination.Name,
				backupConfig.AwsCliPath,
				destinationConfig["endpoint_url"].(string),
				destinationConfig["access_key_id"].(string),
				destinationConfig["secret_access_key"].(string),
				basePath,
			))
		case "scp":
			basePath := destinationConfig["destination"].(string)
			backupers = append(backupers, scp.New(
				destination.Name,
				destinationConfig["server"].(string),
				destinationConfig["port"].(int),
				destinationConfig["user"].(string),
				destinationConfig["key"].(string),
				destinationConfig["fingerprint"].(string),
				basePath,
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
				os.Getenv("GCP_SERVICE_ACCOUNT_FILE"),
				destinationConfig["project_id"].(string),
				destinationConfig["bucket_name"].(string),
			))
		default:
			logger.Error(fmt.Sprintf("Unknown destination type: %s", destination.DestType), nil)
			os.Exit(2)
		}
	}

	return backupers
}
