// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package scp

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/process"
)

type SCPClient struct {
	name         string
	host         string
	port         int
	username     string
	privateKey   string
	fingerPrint  string
	remotePathFn func() string
	SCPCommand   string
	SSHCommand   string
}

func New(name, host string, port int, username, privateKeyPath, fingerPrint string, remotePathFn func() string) *SCPClient {
	return &SCPClient{
		name:         name,
		host:         host,
		port:         port,
		username:     username,
		privateKey:   privateKeyPath,
		fingerPrint:  fingerPrint,
		remotePathFn: remotePathFn,
		SCPCommand:   "scp",
		SSHCommand:   "ssh",
	}
}

func (client *SCPClient) generateBackupKey() (string, error) {
	privateKeyFile, err := ioutil.TempFile("", "backup_key")
	if err != nil {
		return "", err
	}
	privateKeyFile.WriteString(client.privateKey)
	privateKeyFile.Close()
	privateKeyFile.Chmod(0400)
	return privateKeyFile.Name(), nil
}

func (client *SCPClient) generateKnownHosts(sessionLogger lager.Logger) (string, error) {
	knownHostsContent := ""
	if client.fingerPrint == "" {
		sessionLogger.Info("Fingerprint not found, performing key-scan")
		cmd := exec.Command("ssh-keyscan", "-p", strconv.Itoa(client.port), client.host)
		sshKeyscanOutput, err := cmd.CombinedOutput()
		if err != nil {
			wrappedErr := fmt.Errorf("error performing ssh-keyscan: '%s', output: '%s'", err, sshKeyscanOutput)
			sessionLogger.Error("scp", wrappedErr)
			return "", wrappedErr
		}
		knownHostsContent = string(sshKeyscanOutput)
	} else {
		knownHostsContent = client.fingerPrint
	}

	knownHostsFile, err := ioutil.TempFile("", "known_hosts")
	if err != nil {
		return "", err
	}
	knownHostsFile.WriteString(knownHostsContent)
	knownHostsFile.Close()
	return knownHostsFile.Name(), nil
}

func (client *SCPClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	privateKeyFileName, err := client.generateBackupKey()
	if err != nil {
		return err
	}

	knownHostsFileName, err := client.generateKnownHosts(sessionLogger)
	if err != nil {
		return err
	}

	defer os.Remove(privateKeyFileName)

	remotePath := client.remotePathFn()

	if err = client.ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName, sessionLogger); err != nil {
		return err
	}

	scpDest := fmt.Sprintf("%s@%s:%s", client.username, client.host, remotePath)
	cmd := exec.Command(client.SCPCommand, "-oStrictHostKeyChecking=yes", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-P", strconv.Itoa(client.port), "-r", ".", scpDest)

	cmd.Dir = localPath
	scpCommandOutput, err := processManager.Start(cmd)
	if err != nil {
		wrappedErr := fmt.Errorf("error performing SCP: '%s', output: '%s'", err, scpCommandOutput)
		sessionLogger.Error("scp", wrappedErr)
		return wrappedErr
	}

	sessionLogger.Info("scp completed")
	return nil
}

func (client *SCPClient) ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName string, sessionLogger lager.Logger) error {
	cmd := exec.Command(client.SSHCommand, "-oStrictHostKeyChecking=yes", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-p", fmt.Sprintf("%d", client.port),
		fmt.Sprintf("%s@%s", client.username, client.host),
		fmt.Sprintf("mkdir -p %s", remotePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error checking if remote path exists: '%s', output: '%s'", err, output)
		sessionLogger.Error("ssh", wrappedErr)
		return wrappedErr
	}

	return nil
}

func (c *SCPClient) Name() string {
	return c.name
}
