package backup

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-golang/lager"
)

type Executor interface {
	RunOnce() error
}

type backup struct {
	awsCLIBinaryPath   string
	sourceFolder       string
	destFolder         string
	awsAccessKeyID     string
	awsSecretAccessKey string
	endpointURL        string
	backupCreatorCmd   string
	cleanupCmd         string
	logger             lager.Logger
}

func NewExecutor(
	awsCLIBinaryPath,
	sourceFolder,
	destFolder,
	awsAccessKeyID,
	awsSecretAccessKey,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd string,
	logger lager.Logger,
) Executor {
	return &backup{
		awsCLIBinaryPath:   awsCLIBinaryPath,
		sourceFolder:       sourceFolder,
		destFolder:         destFolder,
		awsAccessKeyID:     awsAccessKeyID,
		awsSecretAccessKey: awsSecretAccessKey,
		endpointURL:        endpointURL,
		backupCreatorCmd:   backupCreatorCmd,
		cleanupCmd:         cleanupCmd,
		logger:             logger,
	}
}

func (b *backup) performBackup() error {
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("performBackup", lager.Data{"cmd": b.backupCreatorCmd, "out": string(out)})

	return err
}

func (b *backup) performCleanup() error {
	if b.cleanupCmd == "" {
		b.logger.Info("Cleanup command not provided")
		return nil
	}

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("performCleanup", lager.Data{"cmd": b.cleanupCmd, "out": string(out)})

	if err != nil {
		return err
	}

	b.logger.Info("Cleanup command successful")
	return nil
}

func (b *backup) uploadBackup() error {
	cmd := exec.Command(
		b.awsCLIBinaryPath,
		"s3",
		"sync",
		b.sourceFolder,
		b.destFolder,
		"--endpoint-url",
		b.endpointURL,
	)

	env := []string{}
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", b.awsAccessKeyID))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", b.awsSecretAccessKey))
	cmd.Env = env

	b.logger.Info("uploadBackup", lager.Data{"command": cmd})

	out, err := cmd.CombinedOutput()
	b.logger.Debug("uploadBackup", lager.Data{"out": string(out)})
	if err != nil {
		return err
	}

	b.logger.Info("backup uploaded ok")
	return nil
}

func (b *backup) RunOnce() error {
	err := b.performBackup()
	if err != nil {
		b.logger.Error("Backup creator command failed", err)
		return err
	}

	err = b.uploadBackup()
	if err != nil {
		b.logger.Error("Backup upload failed", err)
		return err
	}

	err = b.performCleanup()
	if err != nil {
		b.logger.Error("Cleanup command failed", err)
		// Do not return error if cleanup command failed.
	}
	return nil
}
