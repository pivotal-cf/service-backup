// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package azure

import (
	"code.cloudfoundry.org/lager/v3"
	"encoding/base64"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pivotal-cf/service-backup/process"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type AzureClient struct {
	name         string
	accountName  string
	accountKey   string
	container    string
	endpoint     string
	remotePathFn func() string
}

const ChunkSize = 8 * 1024 * 1024 // 8MB
const Limit = 64 * 1024 * 1024    // 64MB

func New(name, accountKey, accountName, container, endpoint string, remotePathFn func() string) *AzureClient {
	return &AzureClient{
		name:         name,
		accountKey:   accountKey,
		accountName:  accountName,
		container:    container,
		endpoint:     endpoint,
		remotePathFn: remotePathFn,
	}
}

func (a *AzureClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	remotePath := a.remotePathFn()

	sessionLogger.Info("Uploading azure blobs", lager.Data{"container": a.container, "localPath": localPath, "remotePath": remotePath})
	sessionLogger.Info("The container and remote path will be created if they don't already exist", lager.Data{"container": a.container, "remotePath": remotePath})
	sessionLogger.Info(fmt.Sprintf("about to upload %s to Azure remote path %s", localPath, remotePath))
	return a.uploadDir(localPath, remotePath, processManager, sessionLogger)
}

func (a *AzureClient) uploadFile(sessionLogger lager.Logger, containerReference *storage.Container, localFilePath, remoteFilePath string) error {
	sessionLogger.Info(fmt.Sprintf("uploadFile: %s to %s", localFilePath, remoteFilePath))
	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("error in uploadFile could not open file: %w", err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error in uploadFile could not get stats of file: %w", err)
	}
	blob := containerReference.GetBlobReference(remoteFilePath)
	// single file limit
	if stat.Size() < Limit {
		// The API will reject requests with size > 256 MiB
		err = blob.CreateBlockBlobFromReader(file, &storage.PutBlobOptions{})
		if err != nil {
			return fmt.Errorf("error in uploadFile could not create block blob from reader: %w", err)
		}
	} else {
		err = blob.CreateBlockBlob(&storage.PutBlobOptions{})
		if err != nil {
			return fmt.Errorf("error in uploadFile cloud not create block blob: %w", err)
		}
		buffer := make([]byte, ChunkSize)
		blocks := []storage.Block{}
		for i := 0; ; i++ {
			bytesRead, err := file.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return fmt.Errorf("error in uploadFile could not read file to buffer: %w", err)
				}
			}
			chunk := buffer[:bytesRead]
			blockID := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("BlockID{%07d}", i)))
			err = blob.PutBlock(blockID, chunk, &storage.PutBlockOptions{})
			if err != nil {
				return fmt.Errorf("error in uploadFile could not put block: %w", err)
			}
			blocks = append(blocks, storage.Block{
				ID:     blockID,
				Status: storage.BlockStatusUncommitted,
			})
		}
		err = blob.PutBlockList(blocks, &storage.PutBlockListOptions{})
		if err != nil {
			return fmt.Errorf("error in uploadFile put list of blocks: %w", err)
		}
	}

	return nil
}

func (a *AzureClient) uploadDir(localFilePath, remoteFileRoot string, processManager process.ProcessManager, sessionLogger lager.Logger) error {
	endpoint := storage.DefaultBaseURL
	if len(a.endpoint) != 0 {
		endpoint = a.endpoint
	}
	azureClient, err := storage.NewClient(a.accountName, a.accountKey, endpoint, storage.DefaultAPIVersion, true)
	if err != nil {
		return fmt.Errorf("error in uploadDir when creating client: %w", err)
	}

	azureBlobService := azureClient.GetBlobService()

	containerReference := azureBlobService.GetContainerReference(a.container)
	_, err = containerReference.CreateIfNotExists(&storage.CreateContainerOptions{})
	if err != nil {
		return fmt.Errorf("error in uploadDir Failed to establish a new connection: %w", err)
	}

	err = filepath.Walk(localFilePath, func(filePath string, d os.FileInfo, err error) error {
		if d.IsDir() {
			return nil
		}

		filePathDifference := strings.Replace(filePath, localFilePath, "", -1)
		remoteFilePath := filepath.Join(remoteFileRoot, filePathDifference)

		return a.uploadFile(sessionLogger, containerReference, filePath, remoteFilePath)
	})
	if err != nil {
		return fmt.Errorf("error in uploadDir when walking dir: %w", err)
	}

	return nil
}

func (a *AzureClient) Name() string {
	return a.name
}
