package parseargs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"

	"gopkg.in/yaml.v2"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/dummy"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
	"github.com/pivotal-golang/lager"
)

//Parse ...
func Parse(osArgs []string) (backup.Executor, *string, lager.Logger) {
	flags := flag.NewFlagSet(osArgs[0], flag.ExitOnError)

	backupConfigPath := osArgs[1]
	var backupConfig = BackupConfig{}
	configYAML, err := ioutil.ReadFile(backupConfigPath)

	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal([]byte(configYAML), &backupConfig)
	if err != nil {
		log.Fatal(err)
	}

	var backupType string
	var destinationConfig map[string]interface{}

	if len(backupConfig.Destinations) == 0 {
		backupType = "skip"
	} else {
		backupType = backupConfig.Destinations[0].DestType
		destinationConfig = backupConfig.Destinations[0].Config
	}
	cf_lager.AddFlags(flags)
	flags.Parse(osArgs[2:])

	logger, _ = cf_lager.New("ServiceBackup")

	exitIfBackupInProgressBooleanValue, err := strconv.ParseBool(backupConfig.ExitIfInProgress)
	if err != nil {
		logger.Fatal("Invalid boolean value for exit_if_in_progress. Please set to true or false.", err)
	}

	var backuper backup.Backuper
	var remotePath string

	switch backupType {
	case "s3":

		remotePath = fmt.Sprintf("%s/%s", destinationConfig["bucket_name"], destinationConfig["bucket_path"])
		backuper = s3.NewCliClient(
			backupConfig.AwsCliPath,
			destinationConfig["endpoint_url"].(string),
			destinationConfig["access_key_id"].(string),
			destinationConfig["secret_access_key"].(string),
			logger,
		)

	case "scp":
		remotePath = destinationConfig["destination"].(string)
		backuper = scp.New(destinationConfig["server"].(string), destinationConfig["port"].(int), destinationConfig["user"].(string), destinationConfig["key"].(string), logger)

	case "azure":
		remotePath = destinationConfig["path"].(string)
		backuper = azure.New(destinationConfig["storage_access_key"].(string), destinationConfig["storage_account"].(string), destinationConfig["container"].(string), destinationConfig["blob_store_base_url"].(string), backupConfig.AzureCliPath, logger)

	case "skip":
		logger.Info("No destination provided - skipping backup")
		dummyExecutor := dummy.NewDummyExecutor(logger)
		// Default cronSchedule to monthly if not provided when destination is also not provided
		// This is needed to successfully run the dummy executor and not exit
		if backupConfig.CronSchedule == "" {
			backupConfig.CronSchedule = "@monthly"
		}
		return dummyExecutor, &backupConfig.CronSchedule, logger

	default:
		logger.Fatal(fmt.Sprintf("Unknown destination type: %s", backupType), nil)
	}

	var calculator = &backup.FileSystemSizeCalculator{}

	executor := backup.NewExecutor(
		backuper,
		backupConfig.SourceFolder,
		remotePath,
		backupConfig.SourceExecutable,
		backupConfig.CleanupExecutable,
		backupConfig.ServiceIdentifierExecutable,
		exitIfBackupInProgressBooleanValue,
		logger,
		exec.Command,
		calculator,
	)

	return executor, &backupConfig.CronSchedule, logger
}

var logger lager.Logger

type destinationType struct {
	DestType string `yaml:"type"`
	Config   map[string]interface{}
}

type BackupConfig struct {
	Destinations                []destinationType `yaml:"destinations"`
	SourceFolder                string            `yaml:"source_folder"`
	SourceExecutable            string            `yaml:"source_executable"`
	CronSchedule                string            `yaml:"cron_schedule"`
	CleanupExecutable           string            `yaml:"cleanup_executable"`
	MissingPropertiesMessage    string            `yaml:"missing_properties_message"`
	ExitIfInProgress            string            `yaml:"exit_if_in_progress"`
	ServiceIdentifierExecutable string            `yaml:"service_identifier_executable"`
	AwsCliPath                  string            `yaml:"aws_cli_path"`
	AzureCliPath                string            `yaml:"azure_cli_path"`
}
