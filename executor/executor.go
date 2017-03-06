package executor

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/backup"
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
	uploader backup.MultiBackuper,
	sourceFolder,
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

type ServiceInstanceError struct {
	error
	ServiceInstanceID string
}

func (e *Executor) backupCanBeStarted() bool {
	e.Lock()
	defer e.Unlock()
	if e.backupInProgress && e.exitIfBackupInProgress {
		return false
	}
	e.backupInProgress = true
	return true
}

func (e *Executor) doneBackup() {
	e.Lock()
	defer e.Unlock()
	e.backupInProgress = false
}

func (e *Executor) RunOnce() error {
	sessionLogger := e.logger.WithData(lager.Data{"backup_guid": uuid.NewV4().String()})

	serviceInstanceID := e.identifyService(sessionLogger)
	if serviceInstanceID != "" {
		sessionLogger = sessionLogger.Session(
			"WithIdentifier",
			lager.Data{"identifier": serviceInstanceID},
		)
	}

	if !e.backupCanBeStarted() {
		err := errors.New("backup operation rejected")
		sessionLogger.Error("Backup currently in progress, exiting. Another backup will not be able to start until this is completed.", err)
		return ServiceInstanceError{
			error:             err,
			ServiceInstanceID: serviceInstanceID,
		}
	}
	defer e.doneBackup()

	if err := e.performBackup(sessionLogger); err != nil {
		return ServiceInstanceError{
			error:             err,
			ServiceInstanceID: serviceInstanceID,
		}
	}

	if err := e.uploadBackup(sessionLogger); err != nil {
		return ServiceInstanceError{
			error:             err,
			ServiceInstanceID: serviceInstanceID,
		}
	}

	// Do not return error if cleanup command failed.
	e.performCleanup(sessionLogger)

	sessionLogger = e.logger

	return nil
}

func (e *Executor) identifyService(sessionLogger lager.Logger) string {
	if e.serviceIdentifierCmd == "" {
		return ""
	}

	args := strings.Split(e.serviceIdentifierCmd, " ")

	_, err := os.Stat(args[0])
	if err != nil {
		sessionLogger.Error("Service identifier command not found", err)
		return ""
	}

	cmd := e.execCommand(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Service identifier command returned error", err)
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (e *Executor) performBackup(sessionLogger lager.Logger) error {
	if e.backupCreatorCmd == "" {
		sessionLogger.Info("source_executable not provided, skipping performing of backup")
		return nil
	}
	sessionLogger.Info("Perform backup started")
	args := strings.Split(e.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	_, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Perform backup completed with error", err)
		return err
	}

	sessionLogger.Info("Perform backup completed successfully")
	return nil
}

func (e *Executor) performCleanup(sessionLogger lager.Logger) error {
	if e.cleanupCmd == "" {
		sessionLogger.Info("Cleanup command not provided")
		return nil
	}
	sessionLogger.Info("Cleanup started")

	args := strings.Split(e.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	_, err := cmd.CombinedOutput()

	if err != nil {
		sessionLogger.Error("Cleanup completed with error", err)
		return err
	}

	sessionLogger.Info("Cleanup completed successfully")
	return nil
}

func (e *Executor) uploadBackup(sessionLogger lager.Logger) error {
	sessionLogger.Info("Upload backup started")

	startTime := time.Now()
	err := e.uploader.Upload(e.sourceFolder, sessionLogger)
	duration := time.Since(startTime)

	if err != nil {
		sessionLogger.Error("Upload backup completed with error", err)
		return err
	}

	size, _ := e.calculator.DirSize(e.sourceFolder)
	sessionLogger.Info("Upload backup completed successfully", lager.Data{
		"duration_in_seconds": duration.Seconds(),
		"size_in_bytes":       size,
	})
	return nil
}
