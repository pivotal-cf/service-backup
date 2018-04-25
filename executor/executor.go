// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package executor

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/upload"
	"github.com/satori/go.uuid"
)

type Executor interface {
	Execute() error
}

type executor struct {
	sync.Mutex
	uploader               upload.Uploader
	sourceFolder           string
	backupCreatorCmd       string
	cleanupCmd             string
	serviceIdentifierCmd   string
	exitIfBackupInProgress bool
	backupInProgress       bool
	logger                 lager.Logger
	processManager         process.ProcessManager
	execCommand            CmdFunc
	dirSize                DirSizeFunc
}

type DirSizeFunc func(string) (int64, error)

type CmdFunc func(string, ...string) *exec.Cmd

func NewExecutor(
	uploader upload.Uploader,
	sourceFolder,
	backupCreatorCmd,
	cleanupCmd,
	serviceIdentifierCmd string,
	exitIfInProgress bool,
	logger lager.Logger,
	processManager process.ProcessManager,
	options ...Option,
) *executor {

	e := &executor{
		uploader:               uploader,
		sourceFolder:           sourceFolder,
		backupCreatorCmd:       backupCreatorCmd,
		cleanupCmd:             cleanupCmd,
		serviceIdentifierCmd:   serviceIdentifierCmd,
		exitIfBackupInProgress: exitIfInProgress,
		backupInProgress:       false,
		logger:                 logger,
		processManager:         processManager,
		execCommand:            exec.Command,
		dirSize:                calculateDirSize,
	}

	for _, opt := range options {
		opt(e)
	}

	return e
}

type ServiceInstanceError struct {
	error
	ServiceInstanceID string
}

func (e *executor) backupCanBeStarted() bool {
	e.Lock()
	defer e.Unlock()

	if e.backupInProgress && e.exitIfBackupInProgress {
		return false
	}
	e.backupInProgress = true
	return true
}

func (e *executor) doneBackup() {
	e.Lock()
	defer e.Unlock()
	e.backupInProgress = false
}

func (e *executor) Execute() error {
	sessionLogger := e.logger.WithData(lager.Data{"backup_guid": uuid.NewV4().String()})

	serviceInstanceID := e.identifyService(sessionLogger)
	if serviceInstanceID != "" {
		sessionLogger = sessionLogger.Session(
			"WithIdentifier",
			lager.Data{"identifier": serviceInstanceID},
		)
	}

	if !e.backupCanBeStarted() {
		errMsg := "Backup currently in progress, exiting. Another backup will not be able to start until this is completed."
		err := errors.New(errMsg)
		sessionLogger.Error(errMsg, err)
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

func (e *executor) identifyService(sessionLogger lager.Logger) string {
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

func (e *executor) performBackup(sessionLogger lager.Logger) error {
	if e.backupCreatorCmd == "" {
		sessionLogger.Info("source_executable not provided, skipping performing of backup")
		return nil
	}
	sessionLogger.Info("Perform backup started")
	args := strings.Split(e.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	_, err := e.processManager.Start(cmd, make(chan struct{}))
	if err != nil {
		sessionLogger.Error("Perform backup completed with error", err)
		return err
	}

	sessionLogger.Info("Perform backup completed successfully")
	return nil
}

func (e *executor) performCleanup(sessionLogger lager.Logger) error {
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

func (e *executor) uploadBackup(sessionLogger lager.Logger) error {
	sessionLogger.Info("Upload backup started")

	startTime := time.Now()
	err := e.uploader.Upload(e.sourceFolder, sessionLogger, e.processManager)
	duration := time.Since(startTime)

	if err != nil {
		sessionLogger.Error("Upload backup completed with error", err)
		return err
	}

	size, _ := e.dirSize(e.sourceFolder)
	sessionLogger.Info("Upload backup completed successfully", lager.Data{
		"duration_in_seconds": duration.Seconds(),
		"size_in_bytes":       size,
	})
	return nil
}
