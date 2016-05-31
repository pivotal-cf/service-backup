package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// Helper ...
type Helper interface {
	DirSize(localPath string) (int64, error)
}

// DefaultHelper ...
type DefaultHelper struct{}

//Backuper ...
type Backuper interface {
	Upload(localPath, remotePath string) error
	SetLogSession(sessionName, sessionIdentifier string)
	CloseLogSession()
}

type backup struct {
	backuper             Backuper
	sourceFolder         string
	remotePath           string
	backupCreatorCmd     string
	cleanupCmd           string
	serviceIdentifierCmd string
	logger               lager.Logger
	sessionLogger        lager.Logger
	execCommand          ExecCommand
	helper               Helper
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
	helper Helper,
) Executor {
	return &backup{
		backuper:             backuper,
		sourceFolder:         sourceFolder,
		remotePath:           remotePath,
		backupCreatorCmd:     backupCreatorCmd,
		cleanupCmd:           cleanupCmd,
		serviceIdentifierCmd: serviceIdentifierCmd,
		logger:               logger,
		sessionLogger:        logger,
		execCommand:          execCommand,
		helper:               helper,
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

	b.sessionLogger = b.logger
	b.backuper.CloseLogSession()
	return nil
}

func (b *backup) identifyService() {
	args := strings.Split(b.serviceIdentifierCmd, " ")

	_, err := os.Stat(args[0])
	if err != nil {
		b.sessionLogger.Error("Service identifier command not found", err)
		return
	}

	cmd := b.execCommand(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		b.sessionLogger.Error("Service identifier command returned error", err)
		return
	}

	sessionName := "WithIdentifier"
	sessionIdentifier := strings.TrimSpace(string(out))

	b.sessionLogger = b.logger.Session(
		sessionName,
		lager.Data{"identifier": sessionIdentifier},
	)
	b.backuper.SetLogSession(sessionName, sessionIdentifier)
}

func (b *backup) performBackup() error {
	b.sessionLogger.Info("Perform backup started")
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.sessionLogger.Debug("Perform backup debug info", lager.Data{"cmd": b.backupCreatorCmd, "out": string(out)})

	if err != nil {
		b.sessionLogger.Error("Perform backup completed with error", err)
		return err
	}

	b.sessionLogger.Info("Perform backup completed without error")
	return nil
}

func (b *backup) performCleanup() error {
	if b.cleanupCmd == "" {
		b.sessionLogger.Info("Cleanup command not provided")
		return nil
	}
	b.sessionLogger.Info("Cleanup started")

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.sessionLogger.Debug("Cleanup debug info", lager.Data{"cmd": b.cleanupCmd, "out": string(out)})

	if err != nil {
		b.sessionLogger.Error("Cleanup completed with error", err)
		return err
	}

	b.sessionLogger.Info("Cleanup completed without error")
	return nil
}

func (b *backup) uploadBackup() error {
	b.sessionLogger.Info("Upload backup started")

	startTime := time.Now()
	err := b.backuper.Upload(
		b.sourceFolder,
		b.remotePathWithDate(),
	)
	duration := time.Since(startTime)

	if err != nil {
		b.sessionLogger.Error("Upload backup completed with error", err)
		return err
	}

	size, err := b.helper.DirSize(b.sourceFolder)
	b.sessionLogger.Info("Upload backup completed without error", lager.Data{
		"duration": duration.Seconds(),
		"size":     size,
	})
	return nil
}

func (b *backup) remotePathWithDate() string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return b.remotePath + "/" + datePath
}

// DirSize ...
func (h *DefaultHelper) DirSize(localPath string) (int64, error) {
	var size int64
	err := filepath.Walk(localPath, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
