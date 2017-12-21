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
)

type AzureClient struct {
	name             string
	accountName      string
	accountKey       string
	container        string
	blobStoreBaseUrl string
	azureCmd         string
	remotePathFn     func() string
}

func New(name, accountKey, accountName, container, blobStoreBaseUrl, azureCmd string, remotePathFn func() string) *AzureClient {
	return &AzureClient{
		name:             name,
		accountKey:       accountKey,
		accountName:      accountName,
		container:        container,
		blobStoreBaseUrl: blobStoreBaseUrl,
		remotePathFn:     remotePathFn,
		azureCmd:         azureCmd,
	}
}

func (a *AzureClient) Upload(localPath string, sessionLogger lager.Logger) error {
	remotePath := a.remotePathFn()

	sessionLogger.Info("Uploading azure blobs", lager.Data{"container": a.container, "localPath": localPath, "remotePath": remotePath})
	sessionLogger.Info("The container and remote path will be created if they don't already exist", lager.Data{"container": a.container, "remotePath": remotePath})
	sessionLogger.Info(fmt.Sprintf("about to upload %s to Azure remote path %s", localPath, remotePath))
	return a.uploadDir(localPath, remotePath, sessionLogger)
}

func (a *AzureClient) uploadDir(localFilePath, remoteFilePath string, sessionLogger lager.Logger) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}

	defer file.Close()

	cmd := exec.Command(
		a.azureCmd, "upload",
		"--local-path", localFilePath,
		"--remote-path", path.Join(a.container, remoteFilePath),
		"--storage-account", a.accountName)

	cmd.Env = append(cmd.Env,
		"LANG="+os.Getenv("LANG"),
		"LC_ALL="+os.Getenv("LC_ALL"),
		"BLOBXFER_STORAGE_ACCOUNT_KEY="+a.accountKey,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		sessionLogger.Info("blobxfer combined output", lager.Data{"output": output})
	}
	return err
}

func (a *AzureClient) Name() string {
	return a.name
}
