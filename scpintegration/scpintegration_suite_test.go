// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package scpintegration_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const (
	sshKeyUsername = "service-backup-tmp-key"
)

var (
	pathToServiceBackupBinary string
	privateKeyPath            string
	privateKeyContents        []byte
	unixUser                  *user.User
)

func createSSHKey() (string, string) {
	sshKeys, err := ioutil.TempDir("", "scp-unit-tests")
	Expect(err).NotTo(HaveOccurred())
	privateKeyPath = filepath.Join(sshKeys, "id_rsa")
	Expect(exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-C", sshKeyUsername,
		"-N", "", "-f", privateKeyPath).Run()).To(Succeed())
	privateKeyContents, err = ioutil.ReadFile(privateKeyPath)
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(sshKeys, "id_rsa.pub"), privateKeyPath
}

func addToAuthorizedKeys(publicKeyPath string) {
	Expect(os.MkdirAll(filepath.Join(unixUser.HomeDir, ".ssh"), 0700)).To(Succeed())
	authKeys, err := os.OpenFile(filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	Expect(err).NotTo(HaveOccurred())
	pubKey, err := os.Open(publicKeyPath)
	Expect(err).NotTo(HaveOccurred())
	defer authKeys.Close()
	defer pubKey.Close()
	_, err = io.Copy(authKeys, pubKey)
	Expect(err).NotTo(HaveOccurred())
}

func removeKeyFromAuthorized() {
	authKeysFilePath := filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys")
	authKeysContent, err := ioutil.ReadFile(authKeysFilePath)
	Expect(err).NotTo(HaveOccurred())

	trimmedAuthKeysLines := [][]byte{}
	for _, line := range bytes.Split(authKeysContent, []byte("\n")) {
		if !bytes.Contains(line, []byte(sshKeyUsername)) {
			trimmedAuthKeysLines = append(trimmedAuthKeysLines, line)
		}
	}

	trimmedAuthKeysContent := bytes.Join(trimmedAuthKeysLines, []byte("\n"))
	err = ioutil.WriteFile(authKeysFilePath, trimmedAuthKeysContent, 0600)
	Expect(err).NotTo(HaveOccurred())
}

var _ = BeforeSuite(func() {
	var publicKeyPath string
	publicKeyPath, privateKeyPath = createSSHKey()

	var err error
	unixUser, err = user.Current()
	Expect(err).NotTo(HaveOccurred())

	addToAuthorizedKeys(publicKeyPath)

	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	removeKeyFromAuthorized()
	Expect(os.RemoveAll(filepath.Dir(privateKeyPath))).To(Succeed())

	gexec.CleanupBuildArtifacts()
})

func TestSCPIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SCP Suite")
}
