package backup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pivotal-golang/lager"
)

//ProviderFactory counterfeiter . ProviderFactory
type ProviderFactory interface {
	ExecCommand(string, ...string) *exec.Cmd
}

//ExecCommand fakeable exec.Command
type ExecCommand func(string, ...string) *exec.Cmd

//Executor ...
type Executor interface {
	RunOnce() error
}

//Backuper ...
type Backuper interface {
	Upload(localPath, remotePath string) error
}

type backup struct {
	backuper             Backuper
	sourceFolder         string
	remotePath           string
	backupCreatorCmd     string
	cleanupCmd           string
	serviceIdentifierCmd string
	logger               lager.Logger
	execCommand          ExecCommand
}

//NewExecutor ...
func NewExecutor(
	backuper Backuper,
	sourceFolder,
	remotePath,
	backupCreatorCmd,
	cleanupCmd,
	serviceIdentifierCmd string,
	logger lager.Logger,
	execCommand ExecCommand,
) Executor {
	return &backup{
		backuper:             backuper,
		sourceFolder:         sourceFolder,
		remotePath:           remotePath,
		backupCreatorCmd:     backupCreatorCmd,
		cleanupCmd:           cleanupCmd,
		serviceIdentifierCmd: serviceIdentifierCmd,
		logger:               logger,
		execCommand:          execCommand,
	}
}

func (b *backup) RunOnce() error {
	if b.serviceIdentifierCmd != "" {
		b.identifyService()
	}

	if err := b.performBackup(); err != nil {
		return err
	}

	if err := b.uploadBackup(); err != nil {
		return err
	}

	// Do not return error if cleanup command failed.
	b.performCleanup()
	return nil
}

func (b *backup) identifyService() {
	args := strings.Split(b.serviceIdentifierCmd, " ")

	_, err := os.Stat(args[0])
	if err != nil {
		b.logger.Error("Service identifier command not found", err)
		return
	}

	cmd := b.execCommand(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		b.logger.Error("Service identifier command returned error", err)
		return
	}

	b.logger = b.logger.Session(
		"",
		lager.Data{"identifier": strings.TrimSpace(string(out))},
	)
}

func (b *backup) performBackup() error {
	b.logger.Info("Perform backup started")
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("Perform backup debug info", lager.Data{"cmd": b.backupCreatorCmd, "out": string(out)})

	if err != nil {
		b.logger.Error("Perform backup completed with error", err)
		return err
	}

	b.logger.Info("Perform backup completed without error")
	return nil
}

func (b *backup) performCleanup() error {
	if b.cleanupCmd == "" {
		b.logger.Info("Cleanup command not provided")
		return nil
	}
	b.logger.Info("Cleanup started")

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("Cleanup debug info", lager.Data{"cmd": b.cleanupCmd, "out": string(out)})

	if err != nil {
		b.logger.Error("Cleanup completed with error", err)
		return err
	}

	b.logger.Info("Cleanup completed without error")
	return nil
}

func (b *backup) uploadBackup() error {
	b.logger.Info("Upload backup started")

	err := b.backuper.Upload(
		b.sourceFolder,
		b.remotePathWithDate(),
	)

	if err != nil {
		b.logger.Error("Upload backup completed with error", err)
		return err
	}

	b.logger.Info("Upload backup completed without error")
	return nil
}

func (b *backup) remotePathWithDate() string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return b.remotePath + "/" + datePath
}
