package backup

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pivotal-golang/lager"
	"github.com/satori/go.uuid"
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
	Upload(localPath string, sessionLogger lager.Logger) error
	Name() string
}

type backup struct {
	sync.Mutex
	uploader               Uploader
	sourceFolder           string
	backupCreatorCmd       string
	cleanupCmd             string
	serviceIdentifierCmd   string
	exitIfBackupInProgress bool
	backupInProgress       bool
	logger                 lager.Logger
	execCommand            ExecCommand
	calculator             SizeCalculator
}

//NewExecutor ...
func NewExecutor(
	uploader Uploader,
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
		uploader:               uploader,
		sourceFolder:           sourceFolder,
		backupCreatorCmd:       backupCreatorCmd,
		cleanupCmd:             cleanupCmd,
		serviceIdentifierCmd:   serviceIdentifierCmd,
		exitIfBackupInProgress: exitIfInProgress,
		backupInProgress:       false,
		logger:                 logger,
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
	sessionLogger := b.logger.WithData(lager.Data{"backup_guid": uuid.NewV4().String()})

	if !b.backupCanBeStarted() {
		err := errors.New("backup operation rejected")
		sessionLogger.Error("Backup currently in progress, exiting. Another backup will not be able to start until this is completed.", err)
		return err
	}
	defer b.doneBackup()

	if b.serviceIdentifierCmd != "" {
		sessionLogger = b.identifyService(sessionLogger)
	}

	if err := b.performBackup(sessionLogger); err != nil {
		return err
	}

	if err := b.uploadBackup(sessionLogger); err != nil {
		return err
	}

	// Do not return error if cleanup command failed.
	b.performCleanup(sessionLogger)

	sessionLogger = b.logger

	return nil
}

func (b *backup) identifyService(sessionLogger lager.Logger) lager.Logger {
	args := strings.Split(b.serviceIdentifierCmd, " ")

	_, err := os.Stat(args[0])
	if err != nil {
		sessionLogger.Error("Service identifier command not found", err)
		return sessionLogger
	}

	cmd := b.execCommand(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Service identifier command returned error", err)
		return sessionLogger
	}

	sessionName := "WithIdentifier"
	sessionIdentifier := strings.TrimSpace(string(out))

	return sessionLogger.Session(
		sessionName,
		lager.Data{"identifier": sessionIdentifier},
	)
}

func (b *backup) performBackup(sessionLogger lager.Logger) error {
	if b.backupCreatorCmd == "" {
		sessionLogger.Info("source_executable not provided, skipping performing of backup")
		return nil
	}
	sessionLogger.Info("Perform backup started")
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	sessionLogger.Debug("Perform backup debug info", lager.Data{"cmd": b.backupCreatorCmd, "out": string(out)})

	if err != nil {
		sessionLogger.Error("Perform backup completed with error", err)
		return err
	}

	sessionLogger.Info("Perform backup completed successfully")
	return nil
}

func (b *backup) performCleanup(sessionLogger lager.Logger) error {
	if b.cleanupCmd == "" {
		sessionLogger.Info("Cleanup command not provided")
		return nil
	}
	sessionLogger.Info("Cleanup started")

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	sessionLogger.Debug("Cleanup debug info", lager.Data{"cmd": b.cleanupCmd, "out": string(out)})

	if err != nil {
		sessionLogger.Error("Cleanup completed with error", err)
		return err
	}

	sessionLogger.Info("Cleanup completed successfully")
	return nil
}

func (b *backup) uploadBackup(sessionLogger lager.Logger) error {
	sessionLogger.Info("Upload backup started")

	startTime := time.Now()
	err := b.uploader.Upload(b.sourceFolder, sessionLogger)
	duration := time.Since(startTime)

	if err != nil {
		sessionLogger.Error("Upload backup completed with error", err)
		return err
	}

	size, _ := b.calculator.DirSize(b.sourceFolder)
	sessionLogger.Info("Upload backup completed successfully", lager.Data{
		"duration_in_seconds": duration.Seconds(),
		"size_in_bytes":       size,
	})
	return nil
}
