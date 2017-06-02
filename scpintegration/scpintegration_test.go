// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package scpintegration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("SCP Backup", func() {
	var (
		consistencyThreshold = time.Second * 5
		scpTimeout           = time.Second * 10
	)

	Context("When SCP server is correctly configured with flags", func() {
		var (
			runningBin      *gexec.Session
			baseDir         string
			destPath        string
			hostFingerprint string
			deploymentName  string
		)

		pathWithDate := func(deployment string, endParts ...string) string {
			today := time.Now()
			dateComponents := []string{fmt.Sprintf("%d", today.Year()), fmt.Sprintf("%02d", today.Month()), fmt.Sprintf("%02d", today.Day())}
			args := []string{destPath}
			if deployment != "" {
				args = append(args, deployment)
			}
			args = append(args, dateComponents...)
			args = append(args, endParts...)
			return filepath.Join(args...)
		}

		JustBeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "scp-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			dirToBackup := filepath.Join(baseDir, "source")
			destPath = filepath.Join(baseDir, "target")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())
			Expect(os.Mkdir(destPath, 0755)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			runningBin = performBackup("localhost", unixUser.Username, destPath, string(privateKeyContents), hostFingerprint, 22, dirToBackup, deploymentName)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
		})

		Context("host finger print not provided", func() {
			BeforeEach(func() {
				hostFingerprint = ""
				deploymentName = ""
			})
			It("copies files over SCP", func() {
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("Fingerprint not found, performing key-scan"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("scp completed"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
				runningBin.Terminate().Wait()
				content1, err := ioutil.ReadFile(pathWithDate(deploymentName, "1.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content1).To(Equal([]byte("1")))
				content2, err := ioutil.ReadFile(pathWithDate(deploymentName, "subdir", "2.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content2).To(Equal([]byte("2")))
			})
		})

		Context("valid host finger print provided", func() {
			BeforeEach(func() {
				deploymentName = ""
				cmd := exec.Command("ssh-keyscan", "-p", strconv.Itoa(22), "localhost")
				output, err := cmd.Output()
				Expect(err).NotTo(HaveOccurred())
				hostFingerprint = strings.Split(string(output), "\n")[0]
			})

			It("copies files over SCP", func() {
				Consistently(runningBin.Out, consistencyThreshold).ShouldNot(gbytes.Say("Fingerprint not found, performing key-scan"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("scp completed"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
				runningBin.Terminate().Wait()
				content1, err := ioutil.ReadFile(pathWithDate(deploymentName, "1.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content1).To(Equal([]byte("1")))
				content2, err := ioutil.ReadFile(pathWithDate(deploymentName, "subdir", "2.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content2).To(Equal([]byte("2")))
			})
		})

		Context("invalid host fingerprint provided", func() {
			BeforeEach(func() {
				deploymentName = ""
				hostFingerprint = "localhost ssh-rsa totally-invalid"
			})
			It("fails to copy files over SCP", func() {
				Consistently(runningBin.Out, consistencyThreshold).ShouldNot(gbytes.Say("Fingerprint not found, performing key-scan"))
				Consistently(runningBin.Out, consistencyThreshold).ShouldNot(gbytes.Say("scp completed"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("Host key verification failed"))
				Expect(runningBin.Terminate().Wait().ExitCode()).ToNot(Equal(BeZero()))
			})
		})

		Context("when add_deployment_name_to_backup_path is true", func() {
			BeforeEach(func() {
				hostFingerprint = ""
				deploymentName = "deployment-name"
			})

			It("copies files over SCP with the deployment name in the path", func() {
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("Fingerprint not found, performing key-scan"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say("scp completed"))
				Eventually(runningBin.Out, scpTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
				runningBin.Terminate().Wait()
				content1, err := ioutil.ReadFile(pathWithDate(deploymentName, "1.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content1).To(Equal([]byte("1")))
				content2, err := ioutil.ReadFile(pathWithDate(deploymentName, "subdir", "2.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content2).To(Equal([]byte("2")))
			})
		})
	})
})

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func performBackup(scpServer, scpUser, scpDestination, scpKey, hostFingerprint string, scpPort int, sourceFolder, deploymentName string) *gexec.Session {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	parts := strings.Split(scpKey, "\n")
	scpKey = strings.Join(parts, "\n      ")

	addDeploymentNameToPath := deploymentName != ""

	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: scp
  name: foo
  config:
    server: %s
    user: %s
    destination: %s
    fingerprint: '%s'
    key: |
      %s
    port: %d
source_folder: %s
source_executable: true
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message
deployment_name: %s
add_deployment_name_to_backup_path: %t`, scpServer, scpUser, scpDestination, hostFingerprint, scpKey, scpPort, sourceFolder, deploymentName, addDeploymentNameToPath,
	)))
	file.Close()

	return runBackup(file.Name())
}
