// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package executor_test

import (
	"errors"
	"os/exec"
	"strings"
	"sync"

	"code.cloudfoundry.org/lager/v3"

	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/pivotal-cf/service-backup/process"
	processfakes "github.com/pivotal-cf/service-backup/process/fakes"
)

var _ = Describe("Executor", func() {
	var (
		execCmd        *exec.Cmd
		backupExecutor executor.Executor
		uploader       *fakeUploader
		logger         lager.Logger
		log            *gbytes.Buffer
		processManager *processfakes.FakeProcessManager

		fakeExecArgs [][]string
		fakeExec     = func(name string, args ...string) *exec.Cmd {
			fakeExecArgs = append(fakeExecArgs, append([]string{name}, args...))
			return execCmd
		}
	)

	BeforeEach(func() {
		fakeExecArgs = [][]string{}
		processManager = &processfakes.FakeProcessManager{}

		log = gbytes.NewBuffer()
		logger = lager.NewLogger("executor")
		logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

		uploader = new(fakeUploader)
	})

	Describe("Execute()", func() {
		var (
			executeErr                error
			performIdentifyServiceCmd string
			exitIfBackupInProgress    bool
		)

		BeforeEach(func() {
			performIdentifyServiceCmd = assetPath("fake-service-identifier")
			exitIfBackupInProgress = false
			execCmd = exec.Command("")
		})

		It("starts the backup process", func() {
			backupExecutor = executor.NewExecutor(
				uploader,
				"source-folder",
				assetPath("fake-snapshotter"),
				assetPath("fake-cleanup"),
				"",
				exitIfBackupInProgress,
				logger,
				processManager,
				executor.WithCommandFunc(fakeExec),
			)

			err := backupExecutor.Execute()

			Expect(err).NotTo(HaveOccurred())
			Expect(processManager.StartCallCount()).To(BeNumerically(">", 1))
			command := processManager.StartArgsForCall(0)
			Expect(command.Path).To(Equal(assetPath("fake-snapshotter")))
		})

		It("cleans up after uploading", func() {
			backupExecutor = executor.NewExecutor(
				uploader,
				"source-folder",
				assetPath("fake-snapshotter"),
				assetPath("fake-cleanup"),
				"",
				exitIfBackupInProgress,
				logger,
				processManager,
				executor.WithCommandFunc(fakeExec),
			)

			err := backupExecutor.Execute()

			Expect(err).NotTo(HaveOccurred())

			Expect(processManager.StartCallCount()).To(Equal(2))
			cmd := processManager.StartArgsForCall(1)
			Expect(cmd.Path).To(Equal(assetPath("fake-cleanup")))
		})

		It("propagates errors from the starter", func() {
			backupExecutor = executor.NewExecutor(
				uploader,
				"source-folder",
				"/never/executed/we/use/a/fake",
				assetPath("fake-cleanup"),
				"",
				exitIfBackupInProgress,
				logger,
				processManager,
				executor.WithCommandFunc(fakeExec),
			)

			stubbedError := errors.New("any error")
			processManager.StartReturns([]byte{}, stubbedError)

			err := backupExecutor.Execute()

			Expect(err).To(MatchError("any error"))
		})

		Describe("failures backing up", func() {
			var serviceIdentifierCmd string

			JustBeforeEach(func() {
				backupExecutor = executor.NewExecutor(
					uploader,
					"source-folder",
					"",
					assetPath("fake-cleanup"),
					serviceIdentifierCmd,
					exitIfBackupInProgress,
					logger,
					processManager,
					executor.WithCommandFunc(fakeExec),
				)

				executeErr = backupExecutor.Execute()
			})

			BeforeEach(func() {
				serviceIdentifierCmd = ""
				uploader = &fakeUploader{uploadErr: errors.New("some failure")}
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError("some failure"))
				Expect(executeErr).To(BeAssignableToTypeOf(executor.ServiceInstanceError{}))
				Expect(executeErr.(executor.ServiceInstanceError).ServiceInstanceID).To(Equal(""))
			})

			Context("when the service identifier command is set", func() {
				BeforeEach(func() {
					serviceIdentifierCmd = assetPath("fake-service-identifier")
					execCmd = exec.Command(assetPath("fake-service-identifier"))
				})

				It("returns an error", func() {
					Expect(executeErr).To(MatchError("some failure"))
					Expect(executeErr).To(BeAssignableToTypeOf(executor.ServiceInstanceError{}))
					Expect(executeErr.(executor.ServiceInstanceError).ServiceInstanceID).To(Equal("unit-identifier"))
				})
			})
		})

		Describe("source_executable not provided", func() {
			JustBeforeEach(func() {
				backupExecutor = executor.NewExecutor(
					uploader,
					"source-folder",
					"",
					assetPath("fake-cleanup"),
					"",
					exitIfBackupInProgress,
					logger,
					processManager,
					executor.WithCommandFunc(fakeExec),
				)

				executeErr = backupExecutor.Execute()
			})

			It("should continue with upload", func() {
				Expect(log).To(gbytes.Say("source_executable not provided, skipping performing of backup"))
				Expect(log).To(gbytes.Say("Upload backup started"))
				Expect(log).To(gbytes.Say("Upload backup completed successfully"))
				Expect(log).To(gbytes.Say("Cleanup completed"))
			})

			It("does not return an error", func() {
				Expect(executeErr).ToNot(HaveOccurred())
			})
		})

		Describe("backup_guid", func() {
			JustBeforeEach(func() {
				backupExecutor = executor.NewExecutor(
					uploader,
					"source-folder",
					assetPath("fake-snapshotter"),
					assetPath("fake-cleanup"),
					"",
					exitIfBackupInProgress,
					logger,
					processManager,
					executor.WithCommandFunc(fakeExec),
				)

				executeErr = backupExecutor.Execute()
			})

			It("logs with a guid for the backup", func() {
				Expect(log).To(gbytes.Say(`"backup_guid":`))
			})
		})

		Describe("performIdentifyService", func() {
			JustBeforeEach(func() {
				backupExecutor = executor.NewExecutor(
					uploader,
					"source-folder",
					assetPath("fake-snapshotter"),
					assetPath("fake-cleanup"),
					performIdentifyServiceCmd,
					exitIfBackupInProgress,
					logger,
					processManager,
					executor.WithCommandFunc(fakeExec),
					executor.WithDirSizeFunc(func(string) (int64, error) { return 200, nil }),
				)

				executeErr = backupExecutor.Execute()
			})

			Context("when provided service identifier", func() {
				BeforeEach(func() {
					execCmd = exec.Command(assetPath("fake-service-identifier"))
				})

				It("does not return an error", func() {
					Expect(executeErr).ToNot(HaveOccurred())
				})

				It("makes a system call to service identifier cmd", func() {
					Expect(len(fakeExecArgs)).To(Equal(1))
					Expect(fakeExecArgs[0]).To(Equal([]string{performIdentifyServiceCmd}))
				})

				It("logs with the service identifier", func() {
					Expect(log).To(gbytes.Say("Perform backup started"))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
				})

				It("logs with the identifier for each event", func() {
					Expect(log).To(gbytes.Say("Perform backup started"))
					Expect(log).To(gbytes.Say(`"backup_guid":`))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					Expect(log).To(gbytes.Say("Perform backup completed successfully"))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					Expect(log).To(gbytes.Say("Upload backup started"))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					Expect(log).To(gbytes.Say("Upload backup completed successfully"))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					Expect(log).To(gbytes.Say("Cleanup completed"))
					Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
				})

				It("logs the upload metadata information", func() {
					Expect(log).To(gbytes.Say(`"duration_in_seconds":\d`))
					Expect(log).To(gbytes.Say(`"size_in_bytes":200`))
				})

				Context("service identifier executable returns an error", func() {
					BeforeEach(func() {
						execCmd = exec.Command(assetPath("fake-error-service-identifier"))
					})

					It("does not return an error", func() {
						Expect(executeErr).ToNot(HaveOccurred())
					})

					It("logs that identifier command was unsuccessful", func() {
						Expect(log).To(gbytes.Say("Service identifier command returned error"))
					})

					It("does not log any identifier", func() {
						Expect(log).ToNot(gbytes.Say(`"identifier"`))
					})
				})

				Context("does not exist", func() {
					BeforeEach(func() {
						performIdentifyServiceCmd = "/path/to/nowhere"
					})

					It("does not return an error", func() {
						Expect(executeErr).ToNot(HaveOccurred())
					})

					It("logs that identifier command did not exist", func() {
						Expect(log).To(gbytes.Say("Service identifier command not found"))
					})

					It("does not log any identifier", func() {
						Expect(log).ToNot(gbytes.Say(`"identifier"`))
					})
				})
			})

			Context("when no service identifier command provided", func() {
				BeforeEach(func() {
					performIdentifyServiceCmd = ""
				})

				It("does not return an error", func() {
					Expect(executeErr).ToNot(HaveOccurred())
				})

				It("logs do not mention identifier at all", func() {
					Expect(log).ToNot(gbytes.Say("identifier"))
				})
			})
		})

		Describe("performWithOtherBackupInProgress", func() {
			Context("when exit_if_in_progress is omitted or set to false", func() {
				JustBeforeEach(func() {
					exitIfBackupAlreadyInProgress := false
					backupExecutor = executor.NewExecutor(
						uploader,
						"source-folder",
						assetPath("fake-snapshotter"),
						assetPath("fake-cleanup"),
						performIdentifyServiceCmd,
						exitIfBackupAlreadyInProgress,
						logger,
						processManager,
						executor.WithCommandFunc(fakeExec),
					)
				})

				Context("when a backup is already in progress", func() {
					JustBeforeEach(func() {
						firstBackupErr := backupExecutor.Execute()
						Expect(firstBackupErr).NotTo(HaveOccurred())
					})

					It("starts the upload", func() {
						secondBackupErr := backupExecutor.Execute()
						Expect(secondBackupErr).NotTo(HaveOccurred())
						Expect(len(fakeExecArgs)).To(Equal(2))
						Expect(log).To(gbytes.Say("Upload backup started"))
					})
				})
			})

			Context("when exit_if_in_progress is set to true and a backup is already in progress", func() {

				It("rejects the upload", func() {
					var (
						blockfirstUpload      sync.WaitGroup
						firstBackupInProgress sync.WaitGroup
					)

					blockfirstUpload.Add(1)
					firstBackupInProgress.Add(1)

					uploader = &fakeUploader{
						uploadStub: func(localPath string, _ lager.Logger) error {
							firstBackupInProgress.Done()
							blockfirstUpload.Wait()
							return nil
						},
					}

					backupExecutor = executor.NewExecutor(
						uploader,
						"source-folder",
						assetPath("fake-snapshotter"),
						assetPath("fake-cleanup"),
						performIdentifyServiceCmd,
						true,
						logger,
						processManager,
						executor.WithCommandFunc(fakeExec),
					)

					go func() {
						defer GinkgoRecover()
						firstBackupErr := backupExecutor.Execute()
						Expect(firstBackupErr).NotTo(HaveOccurred())
					}()

					firstBackupInProgress.Wait()
					secondBackupErr := backupExecutor.Execute()
					blockfirstUpload.Done()

					Expect(secondBackupErr).To(MatchError("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
					Expect(strings.Count(string(log.Contents()), "Perform backup started")).To(Equal(1))
					Expect(log.Contents()).To(ContainSubstring("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
				})
			})
		})
	})
})

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}

type fakeUploader struct {
	uploadStub func(string, lager.Logger) error
	uploadErr  error
}

func (f *fakeUploader) Upload(name string, logger lager.Logger, manager process.ProcessManager) error {
	if f.uploadStub != nil {
		return f.uploadStub(name, logger)
	}
	return f.uploadErr
}

func (f *fakeUploader) Name() string { return "fake" }
