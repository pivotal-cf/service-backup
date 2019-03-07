// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package azure

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/process"
)

type AzureClient struct {
	name         string
	accountName  string
	accountKey   string
	container    string
	endpoint     string
	azureCmd     string
	remotePathFn func() string
}

func New(name, accountKey, accountName, container, endpoint, azureCmd string, remotePathFn func() string) *AzureClient {
	return &AzureClient{
		name:         name,
		accountKey:   accountKey,
		accountName:  accountName,
		container:    container,
		endpoint:     endpoint,
		remotePathFn: remotePathFn,
		azureCmd:     azureCmd,
	}
}

func (a *AzureClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	remotePath := a.remotePathFn()

	sessionLogger.Info("Uploading azure blobs", lager.Data{"container": a.container, "localPath": localPath, "remotePath": remotePath})
	sessionLogger.Info("The container and remote path will be created if they don't already exist", lager.Data{"container": a.container, "remotePath": remotePath})
	sessionLogger.Info(fmt.Sprintf("about to upload %s to Azure remote path %s", localPath, remotePath))
	return a.uploadDir(localPath, remotePath, processManager, sessionLogger)
}

func (a *AzureClient) uploadDir(localFilePath, remoteFilePath string, processManager process.ProcessManager, sessionLogger lager.Logger) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	args := []string{
		"upload",
		"--local-path", localFilePath,
		"--remote-path", path.Join(a.container, remoteFilePath),
		"--storage-account", a.accountName}
	if a.endpoint != "" {
		args = append(args, "--endpoint", a.endpoint)
	}
	cmd := exec.Command(a.azureCmd, args...)

	lang := os.Getenv("LANG")
	if lang == "" {
		lang = "C.UTF-8"
	}
	cmd.Env = append(cmd.Env,
		"LANG="+lang,
		"BLOBXFER_STORAGE_ACCOUNT_KEY="+a.accountKey,
	)

	output, err := processManager.Start(cmd)

	if err != nil {
		sessionLogger.Info("blobxfer combined output", lager.Data{"output": string(output)})
	}
	return err
}

func (a *AzureClient) Name() string {
	return a.name
}
