package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/lager"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
)

func Parse(backupConfigPath string, logger lager.Logger) BackupConfig {
	var backupConfig = BackupConfig{}

	configYAML, err := ioutil.ReadFile(backupConfigPath)
	if err != nil {
		logger.Error("Error reading config file", err)
		os.Exit(2)
	}

	err = yaml.Unmarshal([]byte(configYAML), &backupConfig)
	if err != nil {
		logger.Error("Error unmarshalling config file", err)
		os.Exit(2)
	}

	return backupConfig
}

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

func (b BackupConfig) NoDestinations() bool {
	return len(b.Destinations) == 0
}
