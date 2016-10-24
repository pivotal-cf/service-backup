package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/dummy"
	"github.com/pivotal-cf-experimental/service-backup/gcp"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
)

func Parse(backupConfigPath string, logger lager.Logger) (backup.Executor, string, *alerts.ServiceAlertsClient) {
	var backupConfig = BackupConfig{}
	configYAML, err := ioutil.ReadFile(backupConfigPath)

	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal([]byte(configYAML), &backupConfig)
	if err != nil {
		log.Fatal(err)
	}

	alertsClient := parseAlertsClient(backupConfig)

	uploader := backup.Uploader{}

	if len(backupConfig.Destinations) == 0 {
		logger.Info("No destination provided - skipping backup")
		dummyExecutor := dummy.NewDummyExecutor(logger)
		// Default cronSchedule to monthly if not provided when destination is also not provided
		// This is needed to successfully run the dummy executor and not exit
		if backupConfig.CronSchedule == "" {
			backupConfig.CronSchedule = "@monthly"
		}
		return dummyExecutor, backupConfig.CronSchedule, alertsClient
	}

	for _, destination := range backupConfig.Destinations {
		destinationConfig := destination.Config
		switch destination.DestType {
		case "s3":
			basePath := fmt.Sprintf("%s/%s", destinationConfig["bucket_name"], destinationConfig["bucket_path"])
			uploader = append(uploader, s3.New(
				destination.Name,
				backupConfig.AwsCliPath,
				destinationConfig["endpoint_url"].(string),
				destinationConfig["access_key_id"].(string),
				destinationConfig["secret_access_key"].(string),
				basePath,
			))
		case "scp":
			basePath := destinationConfig["destination"].(string)
			uploader = append(uploader, scp.New(
				destination.Name,
				destinationConfig["server"].(string),
				destinationConfig["port"].(int),
				destinationConfig["user"].(string),
				destinationConfig["key"].(string),
				basePath,
				destinationConfig["fingerprint"].(string)))
		case "azure":
			basePath := destinationConfig["path"].(string)
			uploader = append(uploader, azure.New(
				destination.Name,
				destinationConfig["storage_access_key"].(string),
				destinationConfig["storage_account"].(string),
				destinationConfig["container"].(string),
				destinationConfig["blob_store_base_url"].(string),
				backupConfig.AzureCliPath,
				basePath))
		case "gcs":
			uploader = append(uploader, gcp.New(
				os.Getenv("GCP_SERVICE_ACCOUNT_FILE"),
				destinationConfig["project_id"].(string),
				destinationConfig["bucket_name"].(string),
			))
		default:
			logger.Error(fmt.Sprintf("Unknown destination type: %s", destination.DestType), nil)
			os.Exit(2)
		}
	}

	var calculator = &backup.FileSystemSizeCalculator{}

	executor := backup.NewExecutor(
		uploader,
		backupConfig.SourceFolder,
		backupConfig.SourceExecutable,
		backupConfig.CleanupExecutable,
		backupConfig.ServiceIdentifierExecutable,
		backupConfig.ExitIfInProgress,
		logger,
		exec.Command,
		calculator,
	)

	return executor, backupConfig.CronSchedule, alertsClient
}

func parseAlertsClient(backupConfig BackupConfig) *alerts.ServiceAlertsClient {
	if backupConfig.Alerts == nil {
		return nil
	}

	alertsConfig := alerts.Config{
		CloudController:    backupConfig.Alerts.CloudController,
		NotificationTarget: backupConfig.Alerts.NotificationTarget,
	}

	return alerts.New(alertsConfig, nil)
}

var logger lager.Logger

type destinationType struct {
	DestType string `yaml:"type"`
	Name     string `yaml:"name"`
	Config   map[string]interface{}
}

type BackupConfig struct {
	Destinations                []destinationType `yaml:"destinations"`
	SourceFolder                string            `yaml:"source_folder"`
	SourceExecutable            string            `yaml:"source_executable"`
	CronSchedule                string            `yaml:"cron_schedule"`
	CleanupExecutable           string            `yaml:"cleanup_executable"`
	MissingPropertiesMessage    string            `yaml:"missing_properties_message"`
	ExitIfInProgress            bool              `yaml:"exit_if_in_progress"`
	ServiceIdentifierExecutable string            `yaml:"service_identifier_executable"`
	AwsCliPath                  string            `yaml:"aws_cli_path"`
	AzureCliPath                string            `yaml:"azure_cli_path"`
	Alerts                      *struct {
		ProductName        string                    `yaml:"product_name"`
		NotificationTarget alerts.NotificationTarget `yaml:"notification_target"`
		CloudController    alerts.CloudController    `yaml:"cloud_controller"`
	}
}
