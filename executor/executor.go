package executor

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/satori/go.uuid"
)

type Executor struct {
	sync.Mutex
	uploader               backup.MultiBackuper
	sourceFolder           string
	backupCreatorCmd       string
	cleanupCmd             string
	serviceIdentifierCmd   string
	exitIfBackupInProgress bool
	backupInProgress       bool
	logger                 lager.Logger
	execCommand            backup.ExecCommand
	calculator             backup.SizeCalculator
}

func NewExecutor(
	uploader backup.MultiBackuper, sourceFolder,
	backupCreatorCmd,
	cleanupCmd,
	serviceIdentifierCmd string,
	exitIfInProgress bool,
	logger lager.Logger,
	execCommand backup.ExecCommand,
	calculator backup.SizeCalculator,
) backup.Executor {
	return &Executor{
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

func (b *Executor) backupCanBeStarted() bool {
	b.Lock()
	defer b.Unlock()
	if b.backupInProgress && b.exitIfBackupInProgress {
		return false
	}
	b.backupInProgress = true
	return true
}

func (b *Executor) doneBackup() {
	b.Lock()
	defer b.Unlock()
	b.backupInProgress = false
}

func (b *Executor) RunOnce() error {
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

func (b *Executor) identifyService(sessionLogger lager.Logger) lager.Logger {
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

func (b *Executor) performBackup(sessionLogger lager.Logger) error {
	if b.backupCreatorCmd == "" {
		sessionLogger.Info("source_executable not provided, skipping performing of backup")
		return nil
	}
	sessionLogger.Info("Perform backup started")
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	_, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Perform backup completed with error", err)
		return err
	}

	sessionLogger.Info("Perform backup completed successfully")
	return nil
}

func (b *Executor) performCleanup(sessionLogger lager.Logger) error {
	if b.cleanupCmd == "" {
		sessionLogger.Info("Cleanup command not provided")
		return nil
	}
	sessionLogger.Info("Cleanup started")

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	_, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Cleanup completed with error", err)
		return err
	}

	sessionLogger.Info("Cleanup completed successfully")
	return nil
}

func (b *Executor) uploadBackup(sessionLogger lager.Logger) error {
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
