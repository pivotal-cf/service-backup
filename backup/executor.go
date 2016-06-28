package backup

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o backupfakes/fake_provider_factory.go . ProviderFactory
type ProviderFactory interface {
	ExecCommand(string, ...string) *exec.Cmd
}

//ExecCommand fakeable exec.Command
type ExecCommand func(string, ...string) *exec.Cmd

type Executor interface {
	RunOnce() error
}

//go:generate counterfeiter -o backupfakes/fake_backuper.go . Backuper
type Backuper interface {
	Upload(localPath string) error
	SetLogSession(sessionName, sessionIdentifier string)
	CloseLogSession()
}

type backup struct {
	sync.Mutex
	backuper               Backuper
	sourceFolder           string
	backupCreatorCmd       string
	cleanupCmd             string
	serviceIdentifierCmd   string
	exitIfBackupInProgress bool
	backupInProgress       bool
	logger                 lager.Logger
	sessionLogger          lager.Logger
	execCommand            ExecCommand
	calculator             SizeCalculator
}

//NewExecutor ...
func NewExecutor(
	backuper Backuper,
	sourceFolder,
	backupCreatorCmd,
	cleanupCmd,
	serviceIdentifierCmd string,
	exitIfInProgress bool,
	logger lager.Logger,
	execCommand ExecCommand,
	calculator SizeCalculator,
) Executor {
	return &backup{
		backuper:               backuper,
		sourceFolder:           sourceFolder,
		backupCreatorCmd:       backupCreatorCmd,
		cleanupCmd:             cleanupCmd,
		serviceIdentifierCmd:   serviceIdentifierCmd,
		exitIfBackupInProgress: exitIfInProgress,
		backupInProgress:       false,
		logger:                 logger,
		sessionLogger:          logger,
		execCommand:            execCommand,
		calculator:             calculator,
	}
}

func (b *backup) backupCanBeStarted() bool {
	b.Lock()
	defer b.Unlock()
	if b.backupInProgress && b.exitIfBackupInProgress {
		return false
	}
	b.backupInProgress = true
	return true
}

func (b *backup) doneBackup() {
	b.Lock()
	defer b.Unlock()
	b.backupInProgress = false
}

func (b *backup) RunOnce() error {
	if !b.backupCanBeStarted() {
		err := errors.New("backup operation rejected")
		b.sessionLogger.Error("Backup currently in progress, exiting. Another backup will not be able to start until this is completed.", err)
		return err
	}
	defer b.doneBackup()

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

	b.sessionLogger.Info("Perform backup completed successfully")
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

	b.sessionLogger.Info("Cleanup completed successfully")
	return nil
}

func (b *backup) uploadBackup() error {
	b.sessionLogger.Info("Upload backup started")

	startTime := time.Now()
	err := b.backuper.Upload(b.sourceFolder)
	duration := time.Since(startTime)

	if err != nil {
		b.sessionLogger.Error("Upload backup completed with error", err)
		return err
	}

	size, _ := b.calculator.DirSize(b.sourceFolder)
	b.sessionLogger.Info("Upload backup completed successfully", lager.Data{
		"duration_in_seconds": duration.Seconds(),
		"size_in_bytes":       size,
	})
	return nil
}
