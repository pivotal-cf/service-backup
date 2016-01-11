package backup

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pivotal-golang/lager"
)

type Executor interface {
	RunOnce() error
}

type Backuper interface {
	RemotePathExists(remotePath string) (bool, error)
	CreateRemotePath(remotePath string) error
	Upload(localPath, remotePath string) error
}

type backup struct {
	backuper         Backuper
	sourceFolder     string
	remotePath       string
	backupCreatorCmd string
	cleanupCmd       string
	logger           lager.Logger
}

func NewExecutor(
	backuper Backuper,
	sourceFolder,
	remotePath,
	backupCreatorCmd,
	cleanupCmd string,
	logger lager.Logger,
) Executor {
	return &backup{
		backuper:         backuper,
		sourceFolder:     sourceFolder,
		remotePath:       remotePath,
		backupCreatorCmd: backupCreatorCmd,
		cleanupCmd:       cleanupCmd,
		logger:           logger,
	}
}

func (b *backup) RunOnce() error {
	if err := b.performBackup(); err != nil {
		return err
	}

	if err := b.CreateRemotePathIfNeeded(); err != nil {
		return err
	}

	if err := b.uploadBackup(); err != nil {
		return err
	}

	// Do not return error if cleanup command failed.
	b.performCleanup()
	return nil
}

func (b *backup) CreateRemotePathIfNeeded() error {
	b.logger.Info("Checking for remote path", lager.Data{"remotePath": b.remotePath})
	RemotePathExists, err := b.backuper.RemotePathExists(b.remotePath)
	if err != nil {
		return err
	}

	if RemotePathExists {
		return nil
	}

	b.logger.Info("Checking for remote path - remote path does not exist - making it now")
	err = b.backuper.CreateRemotePath(b.remotePath)
	if err != nil {
		return err
	}
	b.logger.Info("Checking for remote path - remote path created ok")
	return nil
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
